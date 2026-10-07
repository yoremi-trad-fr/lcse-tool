package main

import (
	"bytes"
	"debug/pe"
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func peRawOffset(f *pe.File, rva uint32, length int, data []byte) (int, error) {
	for _, s := range f.Sections {
		if rva >= s.VirtualAddress && uint64(rva-s.VirtualAddress)+uint64(length) <= uint64(s.Size) {
			o := uint64(s.Offset) + uint64(rva-s.VirtualAddress)
			if o+uint64(length) <= uint64(len(data)) {
				return int(o), nil
			}
		}
	}
	return 0, fmt.Errorf("adresse PE hors limites : 0x%X", rva)
}

func peString(f *pe.File, rva uint32, data []byte) (string, error) {
	var out []byte
	for i := uint32(0); i < 512; i++ {
		o, err := peRawOffset(f, rva+i, 1, data)
		if err != nil {
			return "", err
		}
		if data[o] == 0 {
			return string(out), nil
		}
		out = append(out, data[o])
	}
	return "", fmt.Errorf("chaine PE non terminee")
}

func hookExports(data []byte) (map[string]bool, error) {
	f, err := pe.NewFile(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	defer f.Close()
	h, ok := f.OptionalHeader.(*pe.OptionalHeader32)
	if !ok || f.Machine != pe.IMAGE_FILE_MACHINE_I386 || f.Characteristics&pe.IMAGE_FILE_DLL == 0 {
		return nil, fmt.Errorf("DLL du hook 32 bits requise")
	}
	o, err := peRawOffset(f, h.DataDirectory[0].VirtualAddress, 40, data)
	if err != nil {
		return nil, err
	}
	count := binary.LittleEndian.Uint32(data[o+24:])
	if count > 65536 {
		return nil, fmt.Errorf("trop d'exports DLL")
	}
	names, err := peRawOffset(f, binary.LittleEndian.Uint32(data[o+32:]), int(count)*4, data)
	if err != nil {
		return nil, err
	}
	out := map[string]bool{}
	for i := 0; i < int(count); i++ {
		name, err := peString(f, binary.LittleEndian.Uint32(data[names+i*4:]), data)
		if err != nil {
			return nil, err
		}
		out[name] = true
	}
	return out, nil
}

// Redirect GDI imports in an OFFLINE COPY of an unsigned 32-bit game.
// Windows loads lcse_hook.dll through its normal import loader. The entry
// point, machine code and IAT addresses stay intact. The installer saves
// the original separately before replacing a MOON engine in its game folder.
func buildAccentedGame(engine, dll []byte) ([]byte, error) {
	f, err := pe.NewFile(bytes.NewReader(engine))
	if err != nil {
		return nil, fmt.Errorf("moteur PE invalide : %w", err)
	}
	defer f.Close()
	h, ok := f.OptionalHeader.(*pe.OptionalHeader32)
	if !ok || f.Machine != pe.IMAGE_FILE_MACHINE_I386 || f.Characteristics&pe.IMAGE_FILE_DLL != 0 {
		return nil, fmt.Errorf("moteur Windows 32 bits requis")
	}
	if h.NumberOfRvaAndSizes < 16 || h.DataDirectory[4].Size != 0 {
		return nil, fmt.Errorf("moteur signe ou format PE non pris en charge")
	}
	exports, err := hookExports(dll)
	if err != nil {
		return nil, err
	}
	if !exports["GetGlyphOutlineA"] || !exports["CreateFontIndirectA"] {
		return nil, fmt.Errorf("utiliser le kit hook 1.4 (exports GDI absents)")
	}
	if len(engine) < 64 {
		return nil, fmt.Errorf("en-tete PE incomplet")
	}
	nt := int(binary.LittleEndian.Uint32(engine[60:]))
	optional := nt + 24
	sectionHeader := optional + int(f.SizeOfOptionalHeader) + 40*len(f.Sections)
	if sectionHeader+40 > int(h.SizeOfHeaders) || sectionHeader+40 > len(engine) || !bytes.Equal(engine[sectionHeader:sectionHeader+40], make([]byte, 40)) {
		return nil, fmt.Errorf("pas de place pour l'en-tete du module GDI")
	}
	if len(f.Sections) == 0 || len(f.Sections) >= 96 || h.FileAlignment == 0 || h.SectionAlignment == 0 {
		return nil, fmt.Errorf("sections ou alignement PE invalides")
	}
	descriptor := -1
	for i := uint32(0); i < 1024; i++ {
		o, err := peRawOffset(f, h.DataDirectory[1].VirtualAddress+i*20, 20, engine)
		if err != nil {
			return nil, err
		}
		nameRVA := binary.LittleEndian.Uint32(engine[o+12:])
		if nameRVA == 0 {
			break
		}
		name, err := peString(f, nameRVA, engine)
		if err != nil {
			return nil, err
		}
		if strings.EqualFold(name, "lcse_hook.dll") {
			return nil, fmt.Errorf("choisir le moteur original, pas une copie deja modifiee")
		}
		if !strings.EqualFold(name, "gdi32.dll") {
			continue
		}
		if descriptor != -1 {
			return nil, fmt.Errorf("plusieurs tables GDI non prises en charge")
		}
		descriptor = o
		thunk := binary.LittleEndian.Uint32(engine[o:])
		if thunk == 0 {
			thunk = binary.LittleEndian.Uint32(engine[o+16:])
		}
		for j := uint32(0); j < 65536; j++ {
			t, err := peRawOffset(f, thunk+j*4, 4, engine)
			if err != nil {
				return nil, err
			}
			symbol := binary.LittleEndian.Uint32(engine[t:])
			if symbol == 0 {
				break
			}
			if symbol&0x80000000 != 0 {
				return nil, fmt.Errorf("imports GDI par ordinal non pris en charge")
			}
			name, err := peString(f, symbol+2, engine)
			if err != nil {
				return nil, err
			}
			if !exports[name] {
				return nil, fmt.Errorf("fonction GDI absente du kit : %s", name)
			}
		}
	}
	if descriptor < 0 {
		return nil, fmt.Errorf("ce moteur n'importe pas GDI32.dll")
	}
	align := func(v, a uint64) uint64 { return (v + a - 1) / a * a }
	var end uint64
	for _, s := range f.Sections {
		size := uint64(s.VirtualSize)
		if uint64(s.Size) > size {
			size = uint64(s.Size)
		}
		if e := uint64(s.VirtualAddress) + size; e > end {
			end = e
		}
	}
	name := []byte("lcse_hook.dll\x00")
	rva := align(end, uint64(h.SectionAlignment))
	raw := align(uint64(len(engine)), uint64(h.FileAlignment))
	rawSize := align(uint64(len(name)), uint64(h.FileAlignment))
	imageSize := align(rva+uint64(len(name)), uint64(h.SectionAlignment))
	if imageSize > 0xFFFFFFFF || raw+rawSize > maxMoonImageBytes {
		return nil, fmt.Errorf("moteur PE trop grand")
	}
	out := make([]byte, int(raw+rawSize))
	copy(out, engine)
	copy(out[int(raw):], name)
	put := func(o int, v uint32) { binary.LittleEndian.PutUint32(out[o:], v) }
	copy(out[sectionHeader:], ".lcsegdi")
	put(sectionHeader+8, uint32(len(name)))
	put(sectionHeader+12, uint32(rva))
	put(sectionHeader+16, uint32(rawSize))
	put(sectionHeader+20, uint32(raw))
	put(sectionHeader+36, 0x40000040)
	binary.LittleEndian.PutUint16(out[nt+6:], f.NumberOfSections+1)
	put(optional+8, h.SizeOfInitializedData+uint32(rawSize))
	put(optional+56, uint32(imageSize))
	put(descriptor+12, uint32(rva))
	put(descriptor+4, 0)
	put(descriptor+8, 0)
	put(optional+96+11*8, 0)
	put(optional+96+11*8+4, 0) // invalidate bound imports
	put(optional+64, 0)
	var checksum uint64
	for i := 0; i < len(out); i += 2 {
		checksum += uint64(binary.LittleEndian.Uint16(out[i:]))
		checksum = (checksum & 0xFFFF) + (checksum >> 16)
	}
	checksum = (checksum & 0xFFFF) + (checksum >> 16)
	put(optional+64, uint32(checksum)+uint32(len(out)))
	return out, nil
}

func (a *App) installNativeHook(gameDir, engineName, outputName, fontName, debugLog string) string {
	fail := func(err error) string { a.logError(err.Error()); return "ERROR" }
	if gameDir == "" {
		return fail(fmt.Errorf("dossier du jeu requis"))
	}
	if a.oneHookDir == "" {
		a.oneHookDir = a.findDir([]string{"one_hook", "Hook_v5.1"})
	}
	if strings.TrimSpace(fontName) == "" {
		fontName = "MS Gothic"
	}
	if strings.ContainsAny(fontName, "\r\n") || len(fontName) > 31 || (debugLog != "0" && debugLog != "1" && debugLog != "2") {
		return fail(fmt.Errorf("police ou niveau de journal invalide"))
	}
	engine, err := os.ReadFile(filepath.Join(gameDir, engineName))
	if err != nil {
		return fail(err)
	}
	dll, err := os.ReadFile(filepath.Join(a.oneHookDir, "lcse_hook.dll"))
	if err != nil {
		return fail(err)
	}
	patched, err := buildAccentedGame(engine, dll)
	if err != nil {
		return fail(err)
	}
	launcherName := "lcse_launcher.exe"
	launcherPath := filepath.Join(a.oneHookDir, launcherName)
	// The source checkout keeps launchers beside Hook_v5.1; the distributed
	// kit keeps the ONE launcher and DLL together in bin/one_hook.
	if !fileExists(launcherPath) && filepath.Base(a.oneHookDir) == "Hook_v5.1" {
		launcherPath = filepath.Join(filepath.Dir(a.oneHookDir), launcherName)
	}
	launcher, err := os.ReadFile(launcherPath)
	if err != nil {
		return fail(err)
	}
	subdir := "lcse_fr"
	engineTarget := filepath.Join(subdir, engineName)
	dllTarget := filepath.Join(subdir, "lcse_hook.dll")
	iniTarget := filepath.Join(subdir, "lcse_hook.ini")
	content := map[string][]byte{outputName: launcher, engineTarget: patched, dllTarget: dll,
		iniTarget: []byte("[Font]\r\nName=" + fontName + "\r\n\r\n[Debug]\r\nLog=" + debugLog + "\r\n")}
	fontPath := filepath.Join(gameDir, "lcse_font.ttf")
	if font, err := os.ReadFile(fontPath); err == nil {
		content[filepath.Join(subdir, "lcse_font.ttf")] = font
	} else if !os.IsNotExist(err) {
		return fail(err)
	}
	backup := filepath.Join(gameDir, "lcse_backup_"+time.Now().Format("20060102_150405.000000000"))
	old := map[string][]byte{}
	for name := range content {
		path := filepath.Join(gameDir, name)
		data, err := os.ReadFile(path)
		if err == nil {
			old[name] = data
		} else if !os.IsNotExist(err) {
			return fail(err)
		}
	}
	if len(old) > 0 {
		if err := os.Mkdir(backup, 0755); err != nil {
			return fail(err)
		}
		for name, data := range old {
			if err := os.MkdirAll(filepath.Dir(filepath.Join(backup, name)), 0755); err != nil {
				return fail(err)
			}
			if err := os.WriteFile(filepath.Join(backup, name), data, 0644); err != nil {
				return fail(err)
			}
		}
	}
	installed := []string{}
	names := []string{dllTarget, iniTarget, filepath.Join(subdir, "lcse_font.ttf"), engineTarget, outputName}
	for _, name := range names {
		if _, ok := content[name]; !ok {
			continue
		}
		if err := os.MkdirAll(filepath.Dir(filepath.Join(gameDir, name)), 0755); err != nil {
			return fail(err)
		}
	}
	for _, name := range names {
		data, ok := content[name]
		if !ok {
			continue
		}
		if err := os.WriteFile(filepath.Join(gameDir, name), data, 0644); err != nil {
			installed = append(installed, name)
			for _, written := range installed {
				if original, ok := old[written]; ok {
					if restoreErr := os.WriteFile(filepath.Join(gameDir, written), original, 0644); restoreErr != nil {
						a.logError("Restauration : " + restoreErr.Error())
					}
				} else {
					os.Remove(filepath.Join(gameDir, written))
				}
			}
			return fail(err)
		}
		installed = append(installed, name)
	}
	if len(old) > 0 {
		a.log("Sauvegarde : " + backup)
	}
	a.logOK("Accents installes par import GDI normal. Lancer " + outputName + " depuis le dossier du jeu.")
	return "OK"
}
