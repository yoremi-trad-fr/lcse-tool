from pathlib import Path

def main():
    filename = input("Quel fichier voulez-vous extraire ? (ex: DS01.txt) : ").strip()
    
    original_file = Path(filename)
    output_file = original_file.with_name(f"{original_file.stem}_extracted.tsv")

    if not original_file.exists():
        print(f"Erreur : Le fichier '{filename}' est introuvable.")
        return

    # Auto-detect encoding: UTF-8 BOM or Shift-JIS
    raw = original_file.read_bytes()
    if raw[:3] == b'\xef\xbb\xbf':
        encoding = "utf-8-sig"
        print(f"[INFO] Encodage detecte : UTF-8 (BOM)")
    else:
        encoding = "shift_jis"
        print(f"[INFO] Encodage detecte : Shift-JIS -> sortie en UTF-8")

    extracted_data = []

    try:
        content = original_file.read_text(encoding=encoding, errors="replace")
        lines = content.splitlines()
        
        for line in lines:
            if not line or line.startswith("#"):
                continue
            
            parts = line.split("\t", 3)  # max 4 champs: index, offset, type, text
            if len(parts) >= 4 and parts[2] == "TXT":
                extracted_data.append((parts[0], parts[3]))

        with open(output_file, "w", encoding="utf-8-sig", newline="") as f:
            f.write("index\toriginal\n")
            for idx, text in extracted_data:
                f.write(f"{idx}\t{text}\n")

        print(f"Extraction terminee avec succes !")
        print(f"  Lignes TXT extraites : {len(extracted_data)}")
        print(f"  Fichier cree : {output_file}")

    except Exception as e:
        print(f"Une erreur est survenue : {e}")

if __name__ == "__main__":
    main()
