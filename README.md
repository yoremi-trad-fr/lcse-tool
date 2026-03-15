# lcse-tool v0.7

Outil CLI en Go pour le moteur **LC-ScriptEngine** (Nexton).
Developpe pour la traduction francaise de **One ~Kagayaku Kisetsu e~ Vista (2007)**.

## Installation

```
lcse-tool.exe          Outil principal (Windows x64)
lcse-tool              Outil principal (Linux x64)
Extract.py             Extraire les dialogues pour traduction
Reinject.py            Reinjecter les dialogues traduits
Hook/lcse_launcher.exe Lanceur du jeu (charge la police custom)
Hook/lcse_hook.dll     Hook police (espacement proportionnel)
Hook/lcse_hook.ini     Configuration (nom de la police)
Hook/lcse_font.ttf     Police custom (optionnel)
```

## Workflow de traduction

### 1. Preparation (une seule fois)
```bash
lcse-tool unpack lcsebody1 extracted/
lcse-tool snx2txt extracted/ scripts/
```

### 2. Traduction (pour chaque fichier)
```bash
python Extract.py
# -> Entrer le nom du fichier (ex: NV30.txt)
# -> Cree NV30_extracted.tsv (UTF-8 BOM)
# -> Traduire les lignes dans le TSV avec un editeur de texte
python Reinject.py
# -> Entrer le nom du fichier (ex: NV30.txt)
# -> Cree NV30_patched.txt (UTF-8 BOM)
# -> Copier NV30_patched.txt vers scripts/NV30.txt
```

### 3. Injection et patch
```bash
lcse-tool txt2snx-batch scripts/ extracted/ patched/
lcse-tool patch lcsebody1 patched/ lcsebody1_fr
```

### 4. Installation dans le jeu
```
Copier lcsebody1_fr    -> dossier du jeu (renommer en lcsebody1)
Copier lcsebody1_fr.lst -> dossier du jeu (renommer en lcsebody1.lst)
Copier Hook/*          -> dossier du jeu
Lancer via lcse_launcher.exe
```

## Format des fichiers

### scripts/*.txt (UTF-8 BOM, genere par snx2txt)
```
# LCSE SNX: NV30.snx
# INDEX\tOFFSET\tTYPE\tTEXT (UTF-8)
#
0    0x0000    RES    bg_b
3    0x0020    TXT    とても幸せだった…
6    0x009B    TXT    ありがとう、と。
```

### *_extracted.tsv (UTF-8 BOM, genere par Extract.py)
```
index    original
3    とても幸せだった…
6    ありがとう、と。
```
Remplacer le texte japonais par le francais directement dans ce fichier.

## Commandes lcse-tool

```
lcse-tool unpack <lcsebody> [output_dir]
lcse-tool patch <lcsebody_original> <patches_dir> <sortie>
lcse-tool pack <dir> <sortie>
lcse-tool snx2txt <file.snx|dir> [output]
lcse-tool txt2snx <text.txt> <original.snx> [output.snx]
lcse-tool txt2snx-batch <txt_dir> <snx_dir> [output_dir]
```

Options : `--key <hex>` et `--snxkey <hex>` pour forcer les cles XOR.

## Notes techniques

- Les fichiers SNX utilisent des instructions de 12 octets : `[opcode][arg1][arg2]`
- Les references aux chaines sont : `[0x11][0x02][offset_table_chaines]`
- La table de chaines est reconstruite integralement lors de l'injection
- Les fichiers SNX non-standard (NECEMEM) sont copies sans modification
- Les fichiers sans changement sont copies byte-for-byte
- L'encodage est detecte automatiquement (UTF-8 BOM ou Shift-JIS)

## Hook police (optionnel)

Le dossier `Hook/` contient un lanceur + DLL pour charger une police custom.
Cela permet un meilleur espacement des lettres latines.

- `lcse_hook.ini` : configurer le nom de la police
- `lcse_font.ttf` : police custom (doit etre dans le dossier du jeu)
- Lancer le jeu via `lcse_launcher.exe` au lieu de `lcsebody.exe`

## Compilation

```bash
go build -o lcse-tool .
GOOS=windows GOARCH=amd64 go build -o lcse-tool.exe .
```

## Credits

- [GARbro](https://github.com/morkt/GARbro) — format LST/Nexton
- [LCSELocalizationTools](https://github.com/cqjjjzr/LCSELocalizationTools) — outil Java
- [LCScriptEngineTools](https://github.com/fengberd/LCScriptEngineTools) — script PHP
- [The MOON Kit](https://www.asceai.net/moonkit/) — documentation SNX
- lcsebody-main (decompileur Rust) — documentation bytecode 12 octets

## Licence

MIT
