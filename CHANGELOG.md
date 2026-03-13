# CHANGELOG

## v0.4 (2026-03-13) — Bytecode-safe string relocation

### Bugfix critique
- **Corrige le crash du moteur** (`Access violation c0000005` a `lcsebody+0x11d21`)
  cause par la corruption de valeurs entieres dans le bytecode.

### Cause racine
Le bytecode LCSE utilise l'instruction `[0x11] [type] [valeur]` ou `type` determine
l'interpretation de `valeur` :
- `type=0x02` : la valeur est un offset dans la table de chaines (string reference)
- `type=0x00` : la valeur est un entier brut (variable, flag, compteur...)

La v0.3 mettait a jour TOUTE valeur correspondant a un ancien offset de chaine,
sans verifier le type. Des entiers qui avaient par coincidence la meme valeur
numerique qu'un offset etaient corrompus (19 faux positifs sur NV30.snx).
Le moteur tentait ensuite de derefencer ces valeurs comme des pointeurs -> crash.

### Corrections
- **Analyse du bytecode par pattern** : seules les positions verifiant un des
  schemas suivants sont mises a jour :
  - `[0x11] [0x02] [offset]` — instruction "push string"
  - `[0x0F] [offset]` — acces direct par opcode
  - `[0x10] [offset]` — acces direct par opcode
  - `[0x15] [offset]` — acces direct par opcode
- **Ecriture en place** : si la traduction est plus courte ou de meme taille que
  l'original, elle est ecrite directement dans le slot existant de la table de
  chaines, avec zero-padding. Le bytecode n'est pas modifie du tout (zero risque).
- **Relocation en fin de table** : si la traduction deborde, la nouvelle chaine
  est ajoutee en fin de table et seules les references verifiees sont mises a jour.
- **Filet de securite** : le texte japonais original reste a sa position d'origine.
  Si une reference non-identifiee pointe encore vers l'ancien offset, elle
  affichera le texte japonais au lieu de crasher.

### Verification
- Desassemblage de `lcsebody.exe` (RVA 0x11d21) avec Capstone pour confirmer
  le mecanisme de crash (ecriture a un pointeur nul via `mov [edx+edi], cl`)
- Comparaison exhaustive orig vs patched : 0 faux positifs, 0 references manquees

---

## v0.3 (2026-03-10) — Correction des references manquees

### Bugfix
- **Corrige 21 references manquees** dans le bytecode de NV30.snx qui causaient
  un crash au lancement.

### Cause
La v0.2 ne mettait a jour que les offsets precedes des opcodes `0x02` ou `0x10`.
En realite, d'autres opcodes (`0x0F`, `0x00` apres `0x11`, `0x15`) referencent
aussi la table de chaines.

### Modification
- Suppression du filtre par opcode : mise a jour de TOUTE valeur correspondant
  a un ancien offset de chaine.

### Probleme
Cette approche trop permissive introduisait des faux positifs (valeurs entieres
corrompues). Corrige en v0.4.

---

## v0.2 (2026-03-10) — Shift-JIS natif, patch mode, format simplifie

### Nouvelles fonctionnalites
- **Commande `patch`** : remplace uniquement les fichiers modifies dans l'archive
  originale, en copiant les fichiers non-modifies (PNG, WAV, etc.) tels quels.
  Resout le probleme de la v0.1 ou `pack` recreait une archive de 5 Mo au lieu
  des 123 Mo de l'original (les 728 PNG etaient omis).
- **Sortie Shift-JIS native** : les fichiers `.txt` exportes sont directement en
  Shift-JIS. Plus de conversion UTF-8 intermediaire, plus de risque de corruption
  lors du re-encodage. Compatible avec Notepad++ (encodage Shift-JIS).
- **Format 4 colonnes** : `INDEX \t OFFSET \t TYPE \t TEXTE`. La traduction se fait
  directement sur la colonne du texte japonais (remplacement in-place).
  Suppression de la 5eme colonne "TRANSLATION" de la v0.1, inutile puisque le
  moteur ne supporte pas le multilangue.

### Corrections
- **Auto-detection de la cle XOR** depuis le 3eme octet du fichier `.lst`
  (meme algorithme que GARbro `OpenNexton`). La cle pour One ~Vista est `0x01`
  (pas `0x02` comme d'autres jeux LCSE).
- **TXT vides** : confirme que les scripts sans texte (BACKLOGNEXT, SKIPBUTTON,
  WINOFF, etc.) sont du pur bytecode — 0 entries dans la table de chaines.

---

## v0.1 (2026-03-09) — Version initiale

### Fonctionnalites
- **`unpack`** : extraction de l'archive `lcsebody1` + `.lst` (XOR dechiffrement
  des noms de fichiers en Shift-JIS, dechiffrement des donnees SNX)
- **`pack`** : recreation d'une archive depuis un dossier de fichiers
- **`snx2txt`** : export des chaines d'un fichier SNX vers un fichier texte UTF-8
  au format TSV 5 colonnes (INDEX, OFFSET, TYPE, ORIGINAL, TRANSLATION)
- **`txt2snx`** : reinjection des traductions depuis le fichier texte vers le SNX,
  avec reconstruction de la table de chaines et mise a jour des offsets bytecode
- **Modes batch** : `snx2txt` sur un dossier, `txt2snx-batch` pour traitement
  en serie
- **Cross-compilation** : binaires Linux et Windows (amd64)

### Reverse-engineering du format SNX
Structure decouverte par analyse des fichiers echantillons (HIKAMI.SNX, DS11.SNX,
CG.SNX) et croisement avec les sources GARbro (`ArcLST.cs`), le script PHP
(`lcse_pack.php` de fengberd), et l'outil Java (`LCSEPackageUtility` de cqjjjzr) :

```
[uint32 code_count]      — nombre d'opcodes dans la section 1
[uint32 str_table_size]  — taille de la table de chaines en octets
[... bytecode ...]       — opcodes uint32 LE (section 1 + section 2)
[... string table ...]   — derniers str_table_size octets du fichier
```

Table de chaines : entrees consecutives `[uint32 longueur] [donnees Shift-JIS] [\0]`.
Les dialogues portent le suffixe `\x02\x03` avant le null terminator.

### Format de l'archive LST
Croise entre les 3 sources existantes et confirme par analyse du `lcsebody1.lst` :

```
[uint32 count XOR key]
Pour chaque entree :
  [uint32 offset XOR key]
  [uint32 size XOR key]
  [0x40 bytes filename XOR key_byte]  (Shift-JIS, null-padded)
  [uint32 type_id]                    (1=snx, 2=bmp, 3=png, 4=wav, 5=ogg)
```

La cle XOR est un seul octet expanse en uint32 (`key | key<<8 | key<<16 | key<<24`).
La cle SNX pour le dechiffrement des donnees est `key + 1`.

### Limitations connues (v0.1)
- Cle XOR codee en dur a `0x02` (incorrect pour One ~Vista qui utilise `0x01`)
- Pas de mode patch (le `pack` recree l'archive de zero, omettant les PNG)
- Sortie en UTF-8 (problemes de conversion avec les caracteres Shift-JIS speciaux)
- Mise a jour des offsets bytecode uniquement apres les opcodes 0x02 et 0x10
  (incomplet, manque 0x0F et 0x15)
- Format 5 colonnes avec colonne TRANSLATION separee (inutile)

---

## References techniques

- **GARbro** ([morkt/GARbro](https://github.com/morkt/GARbro)) — `ArcFormats/Tactics/ArcLST.cs` :
  format LST Nexton, auto-detection de cle, types de fichiers
- **LCSELocalizationTools** ([cqjjjzr](https://github.com/cqjjjzr/LCSELocalizationTools)) —
  outil Java d'extraction/patch, gestion des cles configurables
- **LCScriptEngineTools** ([fengberd](https://github.com/fengberd/LCScriptEngineTools)) —
  script PHP `lcse_pack.php`, reference pour le format d'archive
- **The MOON Kit** ([asceai](https://www.asceai.net/moonkit/)) — documentation du format
  SNX pour MOON.DVD (meme moteur, version anterieure)
- **Desassemblage de lcsebody.exe** — analyse Capstone x86 a RVA 0x11d21 pour
  identifier les patterns d'acces a la table de chaines dans l'interpreteur bytecode
