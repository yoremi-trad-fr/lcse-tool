# lcse-tool v0.4

Outil CLI en Go pour le moteur **LC-ScriptEngine** (Nexton).
Developpe pour la traduction francaise de **One ~Kagayaku Kisetsu e~ Vista (2007)**.

## Compatibilite

| Composant | One ~Vista | Autres jeux LCSE | Notes |
|-----------|-----------|------------------|-------|
| `unpack` / `pack` / `patch` | Teste | Compatible | Cle auto-detectee depuis le .lst |
| `snx2txt` (export) | Teste | Compatible | Le format de la table de chaines est commun |
| `txt2snx` (in-place) | Teste | Compatible | Pas de modification du bytecode |
| `txt2snx` (overflow) | Teste | A verifier | Les patterns d'opcodes ont ete identifies sur One ~Vista |

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
#    Remplacer le texte japonais directement par le francais (sans accents)

# 4. Reinjecter
lcse-tool txt2snx-batch scripts/ extracted/ patched/

# 5. Patcher l'archive originale
lcse-tool patch lcsebody1 patched/ lcsebody1_fr

# 6. Copier lcsebody1_fr + lcsebody1_fr.lst dans le dossier du jeu
```

## Format des fichiers .txt

Fichier TSV en Shift-JIS, 4 colonnes :

```
# LCSE SNX: NV30.snx
# INDEX\tOFFSET\tTYPE\tTEXT (Shift-JIS)
#
0    0x0000    RES    bg_b
3    0x0020    TXT    とても幸せだった…
4    0x0039    TXT    それが日常であることをぼくは...
```

Pour traduire : remplacer le texte japonais directement.
Les accents ne sont pas supportes (police du jeu codee en dur).

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

## Licence

MIT
