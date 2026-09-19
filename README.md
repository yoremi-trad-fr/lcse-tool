# lcse-tools v1.2

Outil CLI en Go pour le moteur **LC-ScriptEngine** (Nexton).
Développé pour les traductions françaises de **One ~Kagayaku Kisetsu e~ Vista
(2007)** et **MOON. DVD**.

## Contenu

```
lcse-tool.exe            Outil principal (Windows x86)
lcse-tool                Outil principal (Linux x64)
Extract.py               Extraire les dialogues pour traduction
Reinject.py              Réinjecter les dialogues traduits
Hook/lcse_launcher.exe   Lanceur ONE (injection DLL)
Hook/moon_launcher.exe   Lanceur MOON (injection DLL)
Hook/lcse_hook.dll       Hook GDI (accents + police)
Hook/lcse_hook.ini       Configuration
Hook/lcse_font.ttf       Police custom (optionnel)
GUI-Sources/             Interface Wails/Svelte pour piloter le workflow
```

## Interface graphique

La GUI se trouve dans `GUI-Sources/` et produit `build/bin/LCSEToolGUI.exe`.
Elle détecte `lcse-tool.exe` à côté du binaire, dans le dossier du dépôt, ou via
sélection manuelle depuis la barre du haut.
Elle embarque aussi un dossier `GUI-Sources/bin/` pour les outils wrapper
utilisés par les workflows ONE/MOON (`lcse-tool.exe`, `moon_asm.exe`,
`moon_extractTGF.exe`, etc.). Les SNX MOON sont désormais désassemblés
directement : le dossier de sources du MOON Kit n'est plus requis. Le kit hook
est placé dans `bin/one_hook/` et peut être installé pour ONE ou MOON depuis
l'onglet **Hook accents**.

Organisation de la GUI :
- **Preparation** : extraction ONE/MOON et generation des TXT.
- **Import/export dialogues** : fichiers `.dlg.txt` pour ONE/MOON.
- **SNX <-> TXT** : conversion dans les deux sens, fichier ou batch.
- **TGF <-> PNG** : extraction/conversion des images MOON vers PNG.
- **Rebuild archive** : patch/pack d'archives sans mention de langue cible.
- **Hook accents** : édition de `lcse_hook.ini` et installation pour ONE/MOON.

```bash
cd GUI-Sources
wails build
```

## Usage

### Préparation (une seule fois)
```bash
lcse-tool unpack lcsebody1 extracted/
lcse-tool snx2txt extracted/ scripts/
```

### Injection et patch
```bash
lcse-tool txt2snx-batch scripts/ extracted/ patched/
lcse-tool patch lcsebody1 patched/ lcsebody1_fr
```
Pour les CG modifiés, les placer dans le dossier `patched/` avant la commande finale.


## Commandes

| Commande | Description |
|----------|-------------|
| `lcse-tool unpack <lcsebody> [output_dir]` | Extraire une archive LST |
| `lcse-tool patch <original> <patches_dir> <out>` | Patcher une archive |
| `lcse-tool pack <dir> <out>` | Créer une archive |
| `lcse-tool snx2txt <file.snx\|dir> [output]` | SNX → TXT (UTF-8 BOM) |
| `lcse-tool txt2snx <text.txt> <orig.snx> [out]` | TXT → SNX |
| `lcse-tool txt2snx-batch <txt/> <snx/> [out/]` | Batch TXT → SNX |
| `lcse-tool moon-accents <snx\|dir> <out>` | Finaliser les accents après `moon_asm` |

Options : `--key <hex>` et `--snxkey <hex>` pour forcer les clés XOR.

### Archives MOON.

`unpack` et `patch` détectent aussi le format LST ancien de **MOON.**
(`moon_jp`, `moon_eng`) : entrées de 44 octets, noms de fichiers avec extension
incluse, clé LST `0xCC`, SNX chiffrés `0xAA` pour `moon_jp` ou en clair pour
`moon_eng`. Les noms du LST sont décodés depuis le CP932/Shift-JIS vers Unicode
à l'extraction, puis réencodés en Shift-JIS lors d'un rebuild afin de préserver
les éventuels noms japonais sous Windows.

```bash
lcse-tool unpack MOON/moon_eng MOON/_tool_extract_eng
lcse-tool patch MOON/moon_eng MOON/_tool_extract_eng MOON/moon_fr
```

Les `.SNX` de MOON. utilisent un bytecode variable plus ancien que celui de
ONE Vista. Depuis la v1.2, `snx2txt` le détecte et le désassemble directement
en UTF-8, y compris dans l'archive anglaise mixte où certains SNX japonais
restent chiffrés par XOR `0xAA`.

La chaîne japonaise de 119 SNX est également prise en charge. `INIT.snx`
contient 93 octets orphelins après son premier `EOF` ; ils sont signalés puis
ignorés, conformément au résultat fourni par le MOON Kit officiel. La GUI
protège aussi les kanji dont le second octet Shift-JIS est `0x5C`, un cas que
`moon_asm.exe` interprète sinon comme un antislash.

L'archive anglaise contient 131 SNX : 125 scripts actifs et les 6 anciens
scripts japonais non scindés (`DAY01`, `DAY02`, `DAY03`, `DAY07T`, `DAY08`,
`DAY20`). La GUI conserve les 131 à l'extraction pour l'audit mais exclut
automatiquement ces 6 résidus du rebuild lorsque leurs variantes A/B existent.

Le [MOON Kit](https://asceai.net/moonkit/) reste la source de `moon_asm.exe`.
Dans la GUI, le workflow MOON est : extraction de l'archive, désassemblage SNX
vers TXT UTF-8, export/import des seuls dialogues, assemblage en SNX, puis
rebuild de l'archive. Les coupures `\n` lues dans un SNX sont traitées comme
des coupures de mise en page ; `moon_asm.exe` les recalcule lors du rebuild.
Le plus gros script japonais reconstruit est à seulement 755 octets de la
limite de 64 Kio. On peut donc traduire depuis les textes japonais, mais il est
plus prudent de réinjecter dans la structure anglaise déjà scindée si la
traduction française devient plus volumineuse.

## Système d'accents français

Les moteurs ONE/MOON ne supportent que le Shift-JIS. Les caractères accentués français
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

Le lanceur ONE (`lcse_launcher.exe`) ou MOON (`moon_launcher.exe`) crée
l'exécutable cible en mode suspendu, injecte `lcse_hook.dll` via
`CreateRemoteThread` + `LoadLibraryA`, puis reprend l'exécution.

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


## Historique des versions

### v1.2 — Chaîne MOON complète
- Désassemblage direct du bytecode SNX ancien, en clair ou XOR `0xAA`
- Validation extraction → TXT → dialogues → import → assemblage sur l'archive anglaise
- Détection des 6 scripts japonais résiduels et rebuild des 125 scripts actifs
- Accents français UTF-8 finalisés après `moon_asm.exe`
- Lanceur/hook d'accents installable pour `MOON_eng.EXE`

### v1.1 — Workflow MOON initial
- Extraction/patch des archives anciennes et conservation des noms Shift-JIS
- Assemblage à partir des sources pré-désassemblées du MOON Kit

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
