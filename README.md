# lcse-tool v0.6.1

Outil CLI en Go pour le moteur **LC-ScriptEngine** (Nexton).
Developpe pour la traduction francaise de **One ~Kagayaku Kisetsu e~ Vista (2007)**.

## Compatibilite

| Composant | One ~Vista | Autres jeux LCSE | Notes |
|-----------|-----------|------------------|-------|
| `unpack` / `pack` / `patch` | Teste | Compatible | Cle auto-detectee depuis le .lst |
| `snx2txt` (export) | Teste | Compatible | Format de la table de chaines commun |
| `txt2snx` (injection) | Teste | Compatible | Scan 12-octets aligne, zero faux positifs |

### Jeux utilisant LC-ScriptEngine (liste non-exhaustive)
MOON.DVD, One ~Kagayaku Kisetsu e~, Mugen Renkan, Kuroinu, et autres titres Tactics/Nexton.
La cle XOR varie par titre (`0x01` pour One ~Vista, `0x02` pour Mugen Renkan, `0xCC` pour MOON.DVD).

## Installation

Binaires pre-compiles dans le zip :
- **Windows** : `lcse-tool.exe`
- **Linux** : `lcse-tool`

## Workflow de traduction

```bash
# 1. Extraire l'archive (cle auto-detectee)
lcse-tool unpack lcsebody1 extracted/

# 2. Exporter les scripts en texte Shift-JIS
lcse-tool snx2txt extracted/ scripts/

# 3. Editer les .txt avec Notepad++ (encodage Shift-JIS)
#    Remplacer le texte japonais directement par le francais

# 4. Reinjecter
lcse-tool txt2snx-batch scripts/ extracted/ patched/

# 5. Patcher l'archive originale
lcse-tool patch lcsebody1 patched/ lcsebody1_fr

# 6. Copier lcsebody1_fr + lcsebody1_fr.lst dans le dossier du jeu
#    (renommer en lcsebody1 + lcsebody1.lst)
```

### Notes importantes
- **Toujours patcher l'archive originale**, jamais une archive deja patchee.
- Les traductions peuvent etre plus longues que l'original : la table de chaines
  est reconstruite automatiquement et les references mises a jour.
- Les fichiers SNX non-standard (comme `NECEMEM.snx`) sont detectes et copies
  sans modification.
- Les fichiers sans changement sont copies byte-for-byte (zero risque).

## Format des fichiers .txt

Fichier TSV en Shift-JIS, 4 colonnes :

```
# LCSE SNX: NV30.snx
# INDEX\tOFFSET\tTYPE\tTEXT (Shift-JIS)
#
0    0x0000    RES    bg_b
1    0x0009    RES    bgchange
3    0x0020    TXT    とても幸せだった…
4    0x0039    TXT    それが日常であることをぼくは...
```

Types :
- **TXT** : dialogue (a traduire)
- **RES** : nom de ressource (ne pas modifier)
- **CTL** : code de controle (ne pas modifier)

## Commandes

```
lcse-tool unpack <lcsebody> [output_dir]
lcse-tool patch <lcsebody_original> <dossier_patches> <sortie>
lcse-tool pack <dossier> <sortie>
lcse-tool snx2txt <fichier.snx|dossier> [sortie]
lcse-tool txt2snx <texte.txt> <original.snx> [sortie.snx]
lcse-tool txt2snx-batch <dossier_txt> <dossier_snx> [dossier_sortie]
```

Options : `--key <hex>` et `--snxkey <hex>` pour forcer les cles XOR.

## Architecture technique

### Format SNX
```
[uint32 instruction_count]   — nombre d'instructions (h0)
[uint32 str_table_size]      — taille de la table de chaines (h1)
[instructions: h0 * 12B]    — bytecode (12 octets par instruction)
[table de chaines: h1 B]    — entrees consecutives [uint32 len][data]
```

Chaque instruction : `[opcode 4B][arg1 4B][arg2 4B]`
- `[0x11][0x02][offset]` = push string (reference a la table)
- `[0x11][0x00][valeur]` = push int (entier brut, ne pas modifier)

### Methode d'injection (v0.6+)
1. Reconstruction de la table de chaines avec les traductions
2. Scan du bytecode a pas de 12 octets (taille d'une instruction)
3. Seules les instructions `[0x11][0x02][X]` sont mises a jour
4. Zero faux positifs garanti par l'alignement sur les frontieres d'instruction

## Compilation

```bash
go build -o lcse-tool .
GOOS=windows GOARCH=amd64 go build -o lcse-tool.exe .
```

Dependance : `golang.org/x/text` (encodage Shift-JIS)

## Credits

- [GARbro](https://github.com/morkt/GARbro) — format LST/Nexton
- [LCSELocalizationTools](https://github.com/cqjjjzr/LCSELocalizationTools) — outil Java
- [LCScriptEngineTools](https://github.com/fengberd/LCScriptEngineTools) — script PHP
- [The MOON Kit](https://www.asceai.net/moonkit/) — documentation SNX
- lcsebody-main (decompileur Rust) — documentation bytecode 12 octets

## Licence

MIT
