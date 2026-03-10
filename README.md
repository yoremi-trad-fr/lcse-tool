# lcse-tool 0.2 [EXPERIMENTAL-en cours de debug]

Outil CLI en Go pour le moteur **LC-ScriptEngine** (Nexton).
Teste avec **One Kagayaku Kisetsu e Vista (2007)**.

## Changements v0.2

- **`patch`** : remplace uniquement les fichiers modifies dans l'archive (garde PNG intacts)
- **Shift-JIS natif** : les .txt sont directement en Shift-JIS (plus de conversion UTF-8)
- **4 colonnes** : INDEX, OFFSET, TYPE, TEXTE - on ecrit la traduction directement sur la colonne du japonais
- **Auto-detection** de la cle XOR depuis le .lst

## Workflow complet

```bash
# 1. Extraire l'archive
lcse-tool unpack lcsebody1 extracted/

# 2. Exporter les scripts en texte (Shift-JIS)
lcse-tool snx2txt extracted/ scripts/

# 3. Editer les .txt avec un editeur Shift-JIS
#    (Notepad++, VS Code avec encodage Shift-JIS)
#    Remplacer le japonais par le francais DIRECTEMENT

# 4. Reinjecter les modifications
lcse-tool txt2snx-batch scripts/ extracted/ patched/

# 5. Patcher l'archive originale (garde les 728 PNG !)
lcse-tool patch lcsebody1 patched/ lcsebody1_fr
```

## Format des fichiers .txt

Fichier TSV en Shift-JIS, 4 colonnes :

```
# LCSE SNX Script: HIKAMI.snx
0	0x0000	RES	bgm15
3	0x0021	TXT	オレはもう誰もいなくなった教室をひとつづつあたってゆく。
29	0x0285	TXT	氷上「メリークリスマス」
```

Pour traduire, remplacer directement le texte japonais :

```
3	0x0021	TXT	Je parcours les salles desertes une par une.
29	0x0285	TXT	Hikami: "Joyeux Noel"
```

- **TXT** = dialogue (a traduire)
- **RES** = nom de ressource (ne pas modifier)
- **CTL** = code de controle (ne pas modifier)

## Commandes

| Commande | Description |
|----------|-------------|
| `unpack <lcsebody> [dir]` | Extraire (cle auto-detectee) |
| `patch <original> <patches> <sortie>` | Remplacer des fichiers dans l'archive |
| `pack <dir> <sortie>` | Recreer une archive de zero |
| `snx2txt <snx\|dir> [sortie]` | SNX -> texte Shift-JIS |
| `txt2snx <txt> <snx> [sortie]` | Reinjecter texte modifie |
| `txt2snx-batch <txt_dir> <snx_dir> [out]` | Batch reinjection |

## Notes

- Les TXT vides (BACKLOGNEXT, SKIPBUTTON, WINOFF...) sont normaux : bytecode pur
- La cle pour One ~Vista est 0x01 (auto-detectee)
- Utiliser un editeur compatible Shift-JIS pour les .txt
- Les accents ne sont PAS supportes (font codee en dur dans l'exe)
