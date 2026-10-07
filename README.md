# lcse-tools v1.4

Outil CLI en Go pour le moteur **LC-ScriptEngine** (Nexton).
Développé pour les traductions françaises de **One ~Kagayaku Kisetsu e~ Vista
(2007)** et **MOON. DVD**.

## Contenu

```
lcse-tool.exe            Outil principal (Windows x86)
lcse-tool                Outil principal (Linux x64)
Extract.py               Extraire les dialogues pour traduction
Reinject.py              Réinjecter les dialogues traduits
Hook/lcse_launcher.exe   Lanceur ONE (demarrage normal)
Hook/MOON_FR.bat         Lanceur MOON avec locale japonaise
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
- **PNG<->TGF** : deux sens explicites, sélection de plusieurs fichiers ou
  d'un dossier. Une archive MOON est proposée uniquement pour TGF -> PNG.
  Les BMP, déjà lisibles après extraction, ne font pas partie de cet onglet.
  Les fichiers existants sont conservés ; choisir un dossier de sortie vide.
  Le codec TGF est natif et ne dépend plus de l'extracteur externe.
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

La sortie de `patch` doit avoir un autre chemin que l'archive source ; l'outil
refuse de l'écraser. Seuls les fichiers dont le nom **et l'extension**
correspondent à une entrée existante sont remplacés. Par exemple, pour modifier
`ATTWIN_LOAD.BMP`, fournir `ATTWIN_LOAD.bmp` : un PNG homonyme est ignoré et
signalé dans le journal.


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

L'installateur 1.4 redirige l'import de `GDI32.dll` vers `lcse_hook.dll` dans
une copie du moteur. Le code machine, le point d'entrée et les adresses des
fonctions importées sont conservés. Windows charge les accents normalement,
sans injection de DLL ni écriture dans un autre processus.

Pour MOON, une sauvegarde datée conserve d'abord l'original puis la copie
remplace `MOON_eng.EXE` dans le dossier du jeu. Ce nom et cet emplacement
permettent au moteur de retrouver `moon_eng` et ses banques sonores.
`lcse_hook_install.json` référence l'original et ses empreintes ; une
réinstallation repart de cet original vérifié. Le dossier `locale/` du jeu
doit être conservé, avec le profil japonais `JAP` sans élévation.
`MOON_FR.bat` lance ce moteur via `LEProc.exe -runas JAP`, vérifie les archives
et les banques sonores, et remplace les anciens lanceurs après sauvegarde.

Pour ONE, la copie reste dans `lcse_fr/` et `lcsebody_fr.exe` la démarre avec
`CreateProcessW`. L'original reste en place.

Windows charge normalement la DLL. Celle-ci exporte les fonctions GDI et
transmet les appels à la bibliothèque système, à l'exception de :

- `CreateFontIndirectA` : applique la police configurée.
- `GetGlyphOutlineA` : rend les 13 accents single-byte avec `GetGlyphOutlineW`.

La configuration et l'éventuelle police privée sont chargées au premier appel
GDI, en dehors de `DllMain`. Le journal est désactivé par défaut.

Pour MOON, lancer uniquement `MOON_FR.bat` ; la configuration est
`lcse_hook.ini` dans le dossier du jeu. Pour ONE, lancer `lcsebody_fr.exe` ;
la configuration est `lcse_fr/lcse_hook.ini`.

Une analyse Defender sans détection reste un résultat local à une date donnée.
Si Microsoft signale encore un fichier, le transmettre comme faux positif via
[Microsoft Security Intelligence](https://www.microsoft.com/en-us/wdsi/filesubmission).

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
; 0 = off, 1 ou 2 = journal police et accents
Log=0
```

Le log de debug (`lcse_hook.log`) permet de tracer les appels
`GetGlyphOutlineA` et de vérifier que les accents sont correctement interceptés.

### Compilation du hook

```bash
i686-w64-mingw32-windres Hook_v5.1/version.rc -o hook-version.o
i686-w64-mingw32-gcc -shared -O2 -Wall -Wextra -o Hook_v5.1/lcse_hook.dll Hook_v5.1/lcse_hook.c Hook_v5.1/lcse_gdi.def hook-version.o -lgdi32 -Wl,--kill-at
i686-w64-mingw32-gcc -O2 -municode -mwindows -o lcse_launcher.exe lcse_launcher.c
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

### v1.4 — PNG/TGF dans les deux sens et hook par import normal
- Codec TGF natif ; choix du sens et sélection multiple de fichiers.
- BMP exclus de l'onglet images ; refus des sources incompatibles et des écrasements.
- Lanceurs classiques et copie séparée du moteur, sans injection de processus.
- 479 TGF comparés octet par octet à l'extracteur officiel ; deux PNG français validés sans perte de pixels.
- 13 accents vérifiés dans leurs formes normales et signées, avec conservation de l'ASCII et du Shift-JIS.

### v1.3 — Images MOON et reconstruction d'archive sécurisée
- Conversion des TGF et BMP MOON vers PNG depuis une archive, un dossier ou un fichier.
- Refus de reconstruire une archive sur elle-même ; erreurs de lecture et d'écriture remontées.
- Noms de l'index MOON conservés exactement, y compris la casse des extensions.
- Fichiers de patch sans entrée correspondante signalés dans le journal.

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
