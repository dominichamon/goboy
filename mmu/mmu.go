package mmu

import (
	"log"
	"os"
)

type mbc struct {
	rombank, rambank, ramon, mode byte
}

var (
	bios = [0x100]byte{
		0x31, 0xFE, 0xFF, 0xAF, 0x21, 0xFF, 0x9F, 0x32, 0xCB, 0x7C, 0x20, 0xFB, 0x21, 0x26, 0xFF, 0x0E,
		0x11, 0x3E, 0x80, 0x32, 0xE2, 0x0C, 0x3E, 0xF3, 0xE2, 0x32, 0x3E, 0x77, 0x77, 0x3E, 0xFC, 0xE0,
		0x47, 0x11, 0x04, 0x01, 0x21, 0x10, 0x80, 0x1A, 0xCD, 0x95, 0x00, 0xCD, 0x96, 0x00, 0x13, 0x7B,
		0xFE, 0x34, 0x20, 0xF3, 0x11, 0xD8, 0x00, 0x06, 0x08, 0x1A, 0x13, 0x22, 0x23, 0x05, 0x20, 0xF9,
		0x3E, 0x19, 0xEA, 0x10, 0x99, 0x21, 0x2F, 0x99, 0x0E, 0x0C, 0x3D, 0x28, 0x08, 0x32, 0x0D, 0x20,
		0xF9, 0x2E, 0x0F, 0x18, 0xF3, 0x67, 0x3E, 0x64, 0x57, 0xE0, 0x42, 0x3E, 0x91, 0xE0, 0x40, 0x04,
		0x1E, 0x02, 0x0E, 0x0C, 0xF0, 0x44, 0xFE, 0x90, 0x20, 0xFA, 0x0D, 0x20, 0xF7, 0x1D, 0x20, 0xF2,
		0x0E, 0x13, 0x24, 0x7C, 0x1E, 0x83, 0xFE, 0x62, 0x28, 0x06, 0x1E, 0xC1, 0xFE, 0x64, 0x20, 0x06,
		0x7B, 0xE2, 0x0C, 0x3E, 0x87, 0xF2, 0xF0, 0x42, 0x90, 0xE0, 0x42, 0x15, 0x20, 0xD2, 0x05, 0x20,
		0x4F, 0x16, 0x20, 0x18, 0xCB, 0x4F, 0x06, 0x04, 0xC5, 0xCB, 0x11, 0x17, 0xC1, 0xCB, 0x11, 0x17,
		0x05, 0x20, 0xF5, 0x22, 0x23, 0x22, 0x23, 0xC9, 0xCE, 0xED, 0x66, 0x66, 0xCC, 0x0D, 0x00, 0x0B,
		0x03, 0x73, 0x00, 0x83, 0x00, 0x0C, 0x00, 0x0D, 0x00, 0x08, 0x11, 0x1F, 0x88, 0x89, 0x00, 0x0E,
		0xDC, 0xCC, 0x6E, 0xE6, 0xDD, 0xDD, 0xD9, 0x99, 0xBB, 0xBB, 0x67, 0x63, 0x6E, 0x0E, 0xEC, 0xCC,
		0xDD, 0xDC, 0x99, 0x9F, 0xBB, 0xB9, 0x33, 0x3E, 0x3c, 0x42, 0xB9, 0xA5, 0xB9, 0xA5, 0x42, 0x4C,
		0x21, 0x04, 0x01, 0x11, 0xA8, 0x00, 0x1A, 0x13, 0xBE, 0x20, 0xFE, 0x23, 0x7D, 0xFE, 0x34, 0x20,
		0xF5, 0x06, 0x19, 0x78, 0x86, 0x23, 0x05, 0x20, 0xFB, 0x86, 0x20, 0xFE, 0x3E, 0x01, 0xE0, 0x50,
	}

	rom      []byte
	carttype byte // TODO: enum?

	romoffs, ramoffs int

	inbios bool
	Ie     byte
	If byte

	mbcs [2]mbc

	eram [32768]byte
	wram [8192]byte
	zram [127]byte
)

func ReadByte(addr int) byte {
	switch addr & 0xF000 {
	// ROM bank 0
	case 0x0000:
		if inbios {
			if addr < 0x0100 {
				return bios[addr]
			} else if addr == 0x0100 {
				inbios = false
				log.Println("mmu: Leaving bios")
			}
		} else {
			return rom[addr]
		}

	case 0x1000, 0x2000, 0x3000:
		return rom[addr]

	// ROM bank 1
	case 0x4000, 0x5000, 0x6000, 0x7000:
		return rom[romoffs+(addr&0x3FFF)]

	// VRAM
	case 0x8000, 0x9000:
		// TODO
		//return gpu.vram[addr & 0x1FFF]
		return 0

	// External RAM
	case 0xA000, 0xB000:
		return eram[ramoffs+(addr&0x1FFF)]

	// Work RAM and echo
	case 0xC000, 0xD000, 0xE000:
		return wram[addr&0x1FFF]

	// Everything else
	case 0xF000:
		switch addr & 0x0F00 {
		// Echo RAM
		case 0x000, 0x100, 0x200, 0x300, 0x400, 0x500, 0x600, 0x700, 0x800, 0x900, 0xA00, 0xB00, 0xC00, 0xD00:
			return wram[addr&0x1FFF]

		// OAM
		case 0xE00:
			if (addr & 0xFF) < 0xA0 {
				// TODO
				//return gpu.oam[addr & 0xFF]
			}
			return 0

		// Zeropage RAM, IO, interrupts
		case 0xF00:
			if addr == 0xFFFF {
				return Ie
			} else if addr > 0xFF7F {
				return zram[addr&0x7F]
			} else {
				switch addr & 0xF0 {
				case 0x00:
					switch addr & 0xF {
					case 0:
						// TODO
						// return key.ReadByte()  // joyp
						return 0
					case 4, 5, 6, 7:
						// TODO
						//return timer.ReadByte(addr)
						return 0
					case 15:
						return If
					default:
						return 0
					}

				case 0x10, 0x20, 0x30:
					return 0

				case 0x40, 0x50, 0x60, 0x70:
					// TODO
					//return gpu.ReadByte(addr)
					return 0
				}
			}
		}
	}
	log.Panic("Failed to read byte from ", addr)
	return 0
}

func ReadWord(addr int) int {
	return int(ReadByte(addr)) + int((ReadByte(addr+1) << 8))
}

func WriteByte(addr int, value byte) {
	switch addr & 0xF000 {
	// ROM bank 0
	// MBC1: turn external RAM on
	case 0x0000, 0x1000:
		if carttype == 1 {
			mbcs[1].ramon = 0
			if (value & 0xF) == 0xA {
				mbcs[1].ramon = 1
			}
		}

	// MBC1: ROM bank switch
	case 0x2000, 0x3000:
		if carttype == 1 {
			mbcs[1].rombank &= 0x60
			value &= 0x1F
			if value == 0 {
				value = 1
			}
			mbcs[1].rombank |= value
			romoffs = int(mbcs[1].rombank) * 0x4000
		}

	// ROM bank 1
	// MBC1: RAM bank switch
	case 0x4000, 0x5000:
		if carttype == 1 {
			if mbcs[1].mode == 0 {
				mbcs[1].rombank &= 0x1F
				mbcs[1].rombank |= ((value & 3) << 5)
				romoffs = int(mbcs[1].rombank) * 0x4000
			} else {
				mbcs[1].rambank = value & 3
				ramoffs = int(mbcs[1].rambank) * 0x2000
			}
		}

	case 0x6000, 0x7000:
		if carttype == 1 {
			mbcs[1].mode = value & 0x1
		}

	// VRAM
	case 0x8000, 0x9000:
		// TODO
		//gpu.vram[addr & 0x1FFF] = value
		//gpu.updatetile(addr & 0x1FFF, value)
		return

	// External RAM
	case 0xA000, 0xB000:
		eram[ramoffs+(addr&0x1FFF)] = value

	// Work RAM and echo
	case 0xC000, 0xD000, 0xE000:
		wram[addr&0x1FFF] = value

	// Everything else
	case 0xF000:
		switch addr & 0x0F00 {
		// Echo RAM
		case 0x000, 0x100, 0x200, 0x300, 0x400, 0x500, 0x600, 0x700, 0x800, 0x900, 0xA00, 0xB00, 0xC00, 0xD00:
			wram[addr&0x1FFF] = value

		// OAM
		case 0xE00:
			if (addr & 0xFF) < 0xA0 {
				// TODO
				//gpu.oam[addr & 0xFF] = value
				//gpu.updateoam(addr, value)
			}

		// Zeropage RAM, IO, interrupts
		case 0xF00:
			if addr == 0xFFFF {
				Ie = value
			} else if addr > 0xFF7F {
				zram[addr&0x7F] = value
			} else {
				switch addr & 0xF0 {
				case 0x00:
					switch addr & 0xF {
					case 0:
						// TODO
						// key.WriteByte(value)  // joyp
					case 4, 5, 6, 7:
						// TODO
						//return timer.WriteByte(addr, value)
					case 15:
						If = value
					}

				case 0x10, 0x20, 0x30:
					return

				case 0x40, 0x50, 0x60, 0x70:
					// TODO
					//gpu.WriteByte(addr, value)
				}
			}
		}
	}
	log.Panic("Failed to write byte to ", addr)
}

func WriteWord(addr, value int) {
	WriteByte(addr, byte(value&0xFF))
	WriteByte(addr+1, byte((value>>8)&0xFF))
}

func Reset() {
	for i := range eram {
		eram[i] = 0
	}
	for i := range wram {
		wram[i] = 0
	}
	for i := range zram {
		zram[i] = 0
	}

	inbios = true
	Ie = 0
	If = 0

	carttype = 0
	mbcs[1].rombank = 0
	mbcs[1].rambank = 0
	mbcs[1].ramon = 0
	mbcs[1].mode = 0

	romoffs = 0x4000
	ramoffs = 0

	log.Println("mmu: Reset")
}

func Boot() {
	inbios = false
}

func Load(file string) {
	f, err := os.Open(file)
	if err != nil {
		panic(err)
	}

	fi, err := f.Stat()
	if err != nil {
		panic(err)
	}

	rom = make([]byte, fi.Size())
	n, err := f.Read(rom)
	if err != nil {
		panic(err)
	}
	if n != int(fi.Size()) {
		log.Panic(n, " read vs ", fi.Size(), " expected")
	}

	carttype = rom[0x0147]

	log.Printf("mmu: ROM %q loaded: %d bytes", file, len(rom))
}
