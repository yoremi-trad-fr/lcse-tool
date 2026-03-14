from pathlib import Path
import csv

def main():
    # Demande le nom du fichier à l'utilisateur
    filename = input("Quel fichier voulez-vous extraire ? (ex: DS01.txt) : ").strip()
    
    original_file = Path(filename)
    
    # On définit le nom de sortie basé sur le nom d'entrée
    output_file = original_file.with_name(f"{original_file.stem}_extracted.tsv")
    encoding = "shift_jis"

    if not original_file.exists():
        print(f"Erreur : Le fichier '{filename}' est introuvable.")
        return

    extracted_data = []

    try:
        # Lecture du fichier source
        content = original_file.read_text(encoding=encoding, errors="replace")
        lines = content.splitlines()
        
        for line in lines:
            if not line or line.startswith("#"):
                continue
            
            parts = line.split("\t")
            # On vérifie si c'est une ligne de texte (TXT)
            if len(parts) >= 4 and parts[2] == "TXT":
                extracted_data.append({
                    "index": parts[0],
                    "original": parts[3]
                })

        # Écriture du fichier TSV
        # On ne met que l'index et l'original (le JAP que tu remplaceras par le FR)
        fieldnames = ["index", "original"]
        with open(output_file, "w", encoding=encoding, newline="", errors="replace") as f:
            writer = csv.DictWriter(f, fieldnames=fieldnames, delimiter="\t")
            writer.writeheader()
            writer.writerows(extracted_data)

        print(f"Extraction terminee avec succes !")
        print(f"Fichier cree : {output_file}")

    except Exception as e:
        print(f"Une erreur est survenue : {e}")

if __name__ == "__main__":
    main()