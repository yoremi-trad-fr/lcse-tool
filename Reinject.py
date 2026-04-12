from pathlib import Path

def detect_encoding(filepath):
    """Detect UTF-8 BOM or Shift-JIS"""
    raw = filepath.read_bytes()
    if raw[:3] == b'\xef\xbb\xbf':
        return "utf-8-sig"
    return "shift_jis"

def main():
    filename = input("Quel fichier voulez-vous patcher ? (ex: DS01.txt) : ").strip()
    
    original_file = Path(filename)
    translation_file = original_file.with_name(f"{original_file.stem}_extracted.tsv")
    output_file = original_file.with_name(f"{original_file.stem}_patched.txt")

    if not original_file.exists():
        print(f"Erreur : Le fichier source '{filename}' est introuvable.")
        return

    if not translation_file.exists():
        print(f"Erreur : Le fichier de traduction '{translation_file.name}' est introuvable.")
        return

    # Detect encodings
    src_enc = detect_encoding(original_file)
    tsv_enc = detect_encoding(translation_file)
    print(f"[INFO] Source : {src_enc}")
    print(f"[INFO] Traduction : {tsv_enc}")

    # Read translations (raw tab-split, skip header)
    translations = {}
    tsv_content = translation_file.read_text(encoding=tsv_enc, errors="replace")
    tsv_lines = tsv_content.splitlines()
    for i, line in enumerate(tsv_lines):
        if i == 0:  # skip header
            continue
        if not line:
            continue
        parts = line.split("\t", 1)
        if len(parts) == 2:
            translations[parts[0]] = parts[1]

    # Read original and build lookup of original texts
    try:
        content = original_file.read_text(encoding=src_enc, errors="replace")
        src_lines = content.splitlines()
    except Exception as e:
        print(f"Erreur lors de la lecture de l'original : {e}")
        return

    originals = {}
    for line in src_lines:
        if not line or line.startswith("#"):
            continue
        p = line.split("\t", 3)
        if len(p) >= 4 and p[2] == "TXT":
            originals[p[0]] = p[3]

    # Clean csv quoting artifact: leading " added by csv.DictWriter
    # but preserve intentional dialogue quotes (paired "...")
    cleaned = 0
    for idx, text in translations.items():
        if idx in originals and text.startswith('"') and not originals[idx].startswith('"'):
            # If text also ends with ", it's intentional dialogue quoting → keep
            if text.endswith('"') and len(text) > 2:
                continue
            translations[idx] = text[1:]
            cleaned += 1
    if cleaned:
        print(f"[INFO] {cleaned} ligne(s) nettoyee(s) (guillemet csv parasite)")

    # Patch
    new_lines = []
    patched_count = 0
    for line in src_lines:
        if not line or line.startswith("#"):
            new_lines.append(line)
            continue
        
        parts = line.split("\t", 3)  # max 4 champs: index, offset, type, text
        if len(parts) >= 4:
            idx = parts[0]
            if parts[2] == "TXT" and idx in translations:
                parts[3] = translations[idx]
                new_lines.append("\t".join(parts))
                patched_count += 1
            else:
                new_lines.append(line)
        else:
            new_lines.append(line)

    # Always output UTF-8 BOM (for lcse-tool txt2snx)
    output_content = "\r\n".join(new_lines) + "\r\n"
    try:
        with open(output_file, "wb") as f:
            f.write(b'\xef\xbb\xbf')  # UTF-8 BOM
            f.write(output_content.encode("utf-8"))
        print(f"Injection terminee avec succes !")
        print(f"  Lignes patchees : {patched_count}")
        print(f"  Fichier cree : {output_file} (UTF-8 BOM)")
    except Exception as e:
        print(f"Erreur lors de l'ecriture du fichier patche : {e}")

if __name__ == "__main__":
    main()
