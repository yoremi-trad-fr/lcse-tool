from pathlib import Path
import csv

def main():
    # Demande le nom du fichier à l'utilisateur
    filename = input("Quel fichier voulez-vous patcher ? (ex: DS01.txt) : ").strip()
    
    original_file = Path(filename)
    # Le script s'attend à trouver le TSV généré par Extract.py
    translation_file = original_file.with_name(f"{original_file.stem}_extracted.tsv")
    # Le fichier de sortie pour ne pas écraser l'original immédiatement (sécurité)
    output_file = original_file.with_name(f"{original_file.stem}_patched.txt")
    
    encoding = "shift_jis"

    if not original_file.exists():
        print(f"Erreur : Le fichier source '{filename}' est introuvable.")
        return

    if not translation_file.exists():
        print(f"Erreur : Le fichier de traduction '{translation_file.name}' est introuvable.")
        return

    translations = {}
    # Lecture des traductions sans modification de contenu
    with open(translation_file, "r", encoding=encoding, errors="replace") as f:
        reader = csv.DictReader(f, delimiter="\t")
        for row in reader:
            idx = row.get("index")
            text = row.get("original") # On récupère le texte tel quel (la colonne modifiée en FR)
            if idx and text is not None:
                translations[idx] = text.strip()

    try:
        content = original_file.read_text(encoding=encoding, errors="replace")
        src_lines = content.splitlines()
    except Exception as e:
        print(f"Erreur lors de la lecture de l'original : {e}")
        return

    new_lines = []
    for line in src_lines:
        if not line or line.startswith("#"):
            new_lines.append(line)
            continue
        
        parts = line.split("\t")
        if len(parts) >= 4:
            idx = parts[0]
            # Si c'est une ligne de texte (TXT) et qu'on a une traduction pour cet index
            if parts[2] == "TXT" and idx in translations:
                # Injection directe du texte sans filtrage
                parts[3] = translations[idx]
                new_lines.append("\t".join(parts))
            else:
                new_lines.append(line)
        else:
            new_lines.append(line)

    # Écriture avec encodage Shift-JIS et fins de lignes Windows (\r\n)
    output_content = "\r\n".join(new_lines) + "\r\n"
    try:
        output_file.write_bytes(output_content.encode(encoding, errors="replace"))
        print(f"Injection terminee avec succes !")
        print(f"Fichier patche cree : {output_file}")
    except Exception as e:
        print(f"Erreur lors de l'ecriture du fichier patche : {e}")

if __name__ == "__main__":
    main()