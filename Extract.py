from pathlib import Path
import csv

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
        out_encoding = "utf-8-sig"
        print(f"[INFO] Encodage detecte : UTF-8 (BOM)")
    else:
        encoding = "shift_jis"
        out_encoding = "utf-8-sig"  # Always output UTF-8 BOM for editing comfort
        print(f"[INFO] Encodage detecte : Shift-JIS -> sortie en UTF-8")

    extracted_data = []

    try:
        content = original_file.read_text(encoding=encoding, errors="replace")
        lines = content.splitlines()
        
        for line in lines:
            if not line or line.startswith("#"):
                continue
            
            parts = line.split("\t")
            if len(parts) >= 4 and parts[2] == "TXT":
                extracted_data.append({
                    "index": parts[0],
                    "original": parts[3]
                })

        fieldnames = ["index", "original"]
        with open(output_file, "w", encoding=out_encoding, newline="", errors="replace") as f:
            writer = csv.DictWriter(f, fieldnames=fieldnames, delimiter="\t")
            writer.writeheader()
            writer.writerows(extracted_data)

        print(f"Extraction terminee avec succes !")
        print(f"  Lignes TXT extraites : {len(extracted_data)}")
        print(f"  Fichier cree : {output_file}")

    except Exception as e:
        print(f"Une erreur est survenue : {e}")

if __name__ == "__main__":
    main()
