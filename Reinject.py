from pathlib import Path
import csv

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

    # Read translations
    translations = {}
    with open(translation_file, "r", encoding=tsv_enc, errors="replace") as f:
        reader = csv.DictReader(f, delimiter="\t")
        for row in reader:
            idx = row.get("index")
            text = row.get("original")
            if idx and text is not None:
                translations[idx] = text.strip()

    # Read original
    try:
        content = original_file.read_text(encoding=src_enc, errors="replace")
        src_lines = content.splitlines()
    except Exception as e:
        print(f"Erreur lors de la lecture de l'original : {e}")
        return

    # Patch
    new_lines = []
    patched_count = 0
    for line in src_lines:
        if not line or line.startswith("#"):
            new_lines.append(line)
            continue
        
        parts = line.split("\t")
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
