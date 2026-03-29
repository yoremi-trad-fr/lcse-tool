# lcse-tools v0.8

Outil CLI en Go pour le moteur **LC-ScriptEngine** (Nexton).
Développé pour la traduction française de **One ~Kagayaku Kisetsu e~ Vista (2007)**.

## Contenu

```
lcse-tool.exe            Outil principal (Windows x86)
lcse-tool                Outil principal (Linux x64)
Extract.py               Extraire les dialogues pour traduction
Reinject.py              Réinjecter les dialogues traduits
Hook/lcse_launcher.exe   Lanceur du jeu (injection DLL)
Hook/lcse_hook.dll       Hook GDI (accents + police)
Hook/lcse_hook.ini       Configuration
Hook/lcse_font.ttf       Police custom (optionnel)
```

## Workflow de traduction

### 1. Préparation (une seule fois)
```bash
lcse-tool unpack lcsebody1 extracted/
lcse-tool snx2txt extracted/ scripts/
```

### 2. Traduction (pour chaque fichier)
```bash
python Extract.py
# -> Entrer le nom du fichier (ex: NV30.txt)
# -> Crée NV30_extracted.tsv (UTF-8 BOM)
# -> Traduire les lignes dans le TSV
python Reinject.py
# -> Entrer le nom du fichier (ex: NV30.txt)
# -> Crée NV30_patched.txt (UTF-8 BOM)
# -> Copier NV30_patched.txt vers scripts/NV30.txt
```

### 3. Injection et patch
```bash
lcse-tool txt2snx-batch scripts/ extracted/ patched/
lcse-tool patch lcsebody1 patched/ lcsebody1_fr
```
Pour les CG modifiés, les placer dans le dossier `patched/` avant la commande finale.

### 4. Installation dans le jeu
```
Copier lcsebody1_fr      -> dossier du jeu (renommer en lcsebody1)
Copier lcsebody1_fr.lst  -> dossier du jeu (renommer en lcsebody1.lst)
Copier Hook/*            -> dossier du jeu
Lancer via lcse_launcher.exe
```

## Commandes

| Commande | Description |
|----------|-------------|
| `lcse-tool unpack <lcsebody> [output_dir]` | Extraire une archive LST |
| `lcse-tool patch <original> <patches_dir> <out>` | Patcher une archive |
| `lcse-tool pack <dir> <out>` | Créer une archive |
| `lcse-tool snx2txt <file.snx\|dir> [output]` | SNX → TXT (UTF-8 BOM) |
| `lcse-tool txt2snx <text.txt> <orig.snx> [out]` | TXT → SNX |
| `lcse-tool txt2snx-batch <txt/> <snx/> [out/]` | Batch TXT → SNX |

Options : `--key <hex>` et `--snxkey <hex>` pour forcer les clés XOR.

## Système d'accents français

Le moteur LCSE ne supporte que le Shift-JIS. Les caractères accentués français
sont encodés dans la plage single-byte half-width katakana (0xA1-0xAD) par
`lcse-tool`, puis interceptés au rendu par `lcse_hook.dll` qui substitue les
vrais glyphes Unicode.

### Mapping

| Byte | Caractère | Unicode |
|------|-----------|---------|
| 0xA1 | é         | U+00E9  |
| 0xA2 | è         | U+00E8  |
| 0xA3 | ç         | U+00E7  |
| 0xA4 | à         | U+00E0  |
| 0xA5 | â         | U+00E2  |
| 0xA6 | û         | U+00FB  |
| 0xA7 | ô         | U+00F4  |
| 0xA8 | ê         | U+00EA  |
| 0xA9 | î         | U+00EE  |
| 0xAA | ù         | U+00F9  |
| 0xAB | ë         | U+00EB  |
| 0xAC | ï         | U+00EF  |
| 0xAD | ü         | U+00FC  |

Les accents majuscules (À, É, Ç...) et les ligatures (œ, Œ) sont réduits à
leur lettre de base en fallback.

### Pourquoi single-byte ?

Le moteur LCSE avance le curseur de rendu en fonction du type de byte :
- **Single-byte** (0x00-0xFF hors lead bytes SJIS) → avance de **12px** (demi-largeur)
- **Double-byte** (lead + trail SJIS) → avance de **24px** (pleine largeur)

Cette avance est codée en dur dans le moteur et ignore les métriques retournées
par `GetGlyphOutlineA`. Encoder les accents en double-byte (F040-F04C, tentative
v0.7) produisait un espacement de 24px pour un glyphe de ~12px de large.

## Hook DLL — Architecture technique

Le lanceur (`lcse_launcher.exe`) crée `lcsebody.exe` en mode suspendu, injecte
`lcse_hook.dll` via `CreateRemoteThread` + `LoadLibraryA`, puis reprend
l'exécution.

### Hooks IAT

La DLL patche l'Import Address Table de l'exécutable pour intercepter deux
fonctions GDI32 :

**`CreateFontIndirectA`** — Substitue le nom de police. Le moteur demande
`ＭＳ ゴシック` (lu depuis INIT.snx entrées 12-15) ; le hook le remplace par
le nom configuré dans `lcse_hook.ini`.

**`GetGlyphOutlineA`** — Point d'interception principal. Le moteur LCSE
n'utilise ni `TextOutA` ni `ExtTextOutA` : il appelle `GetGlyphOutlineA` pour
récupérer le bitmap de chaque glyphe (format `GGO_GRAY4_BITMAP`), puis le
compose lui-même via `BitBlt`. Quand le hook détecte un byte accent (0xA1-0xAD),
il appelle `GetGlyphOutlineW` avec le vrai codepoint Unicode, court-circuitant
le mapping SJIS.

### Sign-extension

Le moteur stocke les caractères dans un `char` signé. Le byte `0xA1` (-95 en
signé) arrive dans `GetGlyphOutlineA` comme `0xFFFFFFA1` après sign-extension
vers `UINT`. Le hook masque avec `& 0xFF` pour détecter correctement la plage
d'accents.

### Configuration (`lcse_hook.ini`)

```ini
[Font]
; MS Gothic fonctionne directement (police Unicode complète)
Name=MS Gothic

[Debug]
; 0 = off, 1 = log accents, 2 = log tous les glyphes
Log=0
```

Le log de debug (`lcse_hook.log`) permet de tracer les appels
`GetGlyphOutlineA` et de vérifier que les accents sont correctement interceptés.

### Compilation du hook

```bash
i686-w64-mingw32-gcc -shared -o lcse_hook.dll lcse_hook.c -lgdi32
i686-w64-mingw32-gcc -o lcse_launcher.exe lcse_launcher.c
```

## Format SNX

Les fichiers `.snx` contiennent le bytecode et les chaînes de texte du moteur.

- **Header** : 8 octets — `[h0: uint32][h1: uint32]`
  - `h0` = nombre d'instructions, `h1` = taille de la table de chaînes
- **Bytecode** : `h0 × 12` octets — instructions de 12 octets `[opcode][arg1][arg2]`
- **Table de chaînes** : séquence de `[len: uint32][data: len octets]`
- Les références aux chaînes sont des instructions `[0x11][0x02][offset]`

Lors de l'injection, la table de chaînes est entièrement reconstruite et toutes
les références dans le bytecode sont mises à jour.

Les fichiers SNX non-standard (ex: `NECEMEM.snx`) sont détectés et copiés sans
modification pour éviter toute corruption.

## Conventions de traduction

- Pas de suffixes honorifiques sauf `senpai`
- `...` pour les ellipses (pas `…`)
- Présent de narration comme temps par défaut
- Style infinitif pour les choix de branches
- Tags moteur `se_` conservés tels quels
- Numéros de ligne jamais modifiés (servent d'offsets binaires)

## Historique des versions

### v0.8 — Accents single-byte + hook GetGlyphOutlineA
- Accents encodés en single-byte (0xA1-0xAD) au lieu de double-byte (F040-F04C)
- Hook `GetGlyphOutlineA` avec correction sign-extension (0xFFFFFFA1)
- Espacement correct (12px) pour les caractères accentués
- MS Gothic comme police par défaut (pas de police custom nécessaire)

### v0.7 — UTF-8 + Font Hook + extraction scripts
- Export UTF-8 BOM, import auto-détection encodage
- Mapping accents vers SJIS user-defined (F040-F04C) — abandonné en v0.8
- Hook `CreateFontIndirectA` pour substitution de police
- Scripts Python Extract/Reinject

### v0.6.1 — Protection fichiers non-standard
- Détection des SNX non-standard (NECEMEM), copie byte-for-byte
- Copie intacte si 0 modifications

### v0.6 — Scan 12-octets aligné
- Reconstruction complète de la table de chaînes
- Scan à pas de 12 octets, zéro faux positifs
- Supprime la limite de taille des traductions

### v0.5-v0.3 — Itérations initiales
- Différentes approches de relocation (faux positifs, crashes)

### v0.2 — Shift-JIS natif, patch mode
- Commande `patch`, auto-détection clé XOR

### v0.1 — Version initiale
- `unpack`, `pack`, `snx2txt`, `txt2snx`
- Reverse-engineering du format SNX et de l'archive LST

## Compilation de lcse-tool

```bash
go build -o lcse-tool .
GOOS=windows GOARCH=386 go build -o lcse-tool.exe .
```

## Références

- [GARbro](https://github.com/morkt/GARbro) — format LST/Nexton
- [LCSELocalizationTools](https://github.com/cqjjjzr/LCSELocalizationTools) — outil Java
- [LCScriptEngineTools](https://github.com/fengberd/LCScriptEngineTools) — script PHP
- [The MOON Kit](https://www.asceai.net/moonkit/) — documentation SNX
- lcsebody-main (décompileur Rust) — documentation bytecode 12 octets

## Licence

MIT
