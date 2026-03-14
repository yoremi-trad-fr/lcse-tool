# CHANGELOG

## v0.6.1 (2026-03-14) — Protection fichiers non-standard + copie intacte

### Bugfix critique
- **Corrige le crash au lancement** cause par la corruption de `NECEMEM.snx`,
  un fichier SNX non-standard qui ne suit pas le format `8 + h0*12 + h1 = taille`.

### Cause racine
Le moteur LCSE charge les fichiers dans l'ordre du `.lst`. Si un seul fichier
change de taille sans raison, tous les offsets suivants de l'archive sont decales
et le moteur lit des donnees corrompues des le demarrage.

`NECEMEM.snx` contient des donnees supplementaires entre le bytecode et la table
de chaines (258 octets non-alignes). La v0.6 tentait de reconstruire sa table,
echouait, et produisait un fichier tronque de 46 octets.

### Corrections
- **Detection des SNX non-standard** : si `8 + h0*12 + h1 != taille_fichier`,
  le fichier est copie byte-for-byte sans modification (`[SKIP]` dans le log).
- **Copie intacte si 0 modifications** : quand `txt2snx` detecte qu'aucune
  chaine n'a ete modifiee, le fichier original est copie tel quel au lieu d'etre
  reconstruit. Elimine tout risque de corruption silencieuse.

---

## v0.6 (2026-03-14) — Scan 12-octets aligne (zero faux positifs)

### Correction fondamentale
- **Supprime la limite de taille des traductions.** La table de chaines est
  entierement reconstruite et les references sont mises a jour de maniere fiable.

### Cause racine des crashs v0.3-v0.5
Le format d'instruction LCSE est de **12 octets** : `[opcode 4B][arg1 4B][arg2 4B]`.
Les versions precedentes scannaient le bytecode a pas de 4 octets, causant des
collisions entre valeurs entieres et references de chaines.

Specifiquement, l'opcode `0x11` ("push literal") utilise `arg1` comme type :
- `arg1 = 0x02` → `arg2` est un **offset dans la table de chaines**
- `arg1 = 0x00` → `arg2` est un **entier brut** (variable, compteur, flag)

Sur NV30.snx, **22 entiers** avaient par coincidence la meme valeur qu'un offset
de chaine. Les versions v0.3-v0.5, qui scannaient a 4 octets, les corrompaient
systematiquement → crash du moteur.

### Solution
- **Scan a pas de 12 octets** (stride = taille d'une instruction).
- Seules les instructions `[0x11][0x02][offset]` a des frontieres d'instruction
  valides sont mises a jour.
- Verification exhaustive sur NV30.snx : **1063 refs trouvees = 1063 entrees**
  dans la table, **0 faux positifs**, **0 references manquees**.

### Source de la decouverte
Reverse-engineering du bytecode LCSE par analyse d'un decompileur Rust
(`lcsebody-main`) dont le fichier `bytecode notes.txt` documente le format
12 octets et les types de l'opcode `0x11`.

---

## v0.5.1 (2026-03-13) — Ecriture en place sans modification du bytecode

### Approche temporaire (abandonnee en v0.6)
- Ecriture strictement en place, sans jamais modifier le bytecode.
- Ajout d'une colonne `MAXLEN` dans le format TXT pour indiquer la taille
  maximale de chaque entree.
- Troncature automatique des traductions trop longues avec avertissement.

### Probleme
548 lignes sur 1063 devaient etre tronquees (le francais est ~28% plus gros
que le japonais en Shift-JIS). Approche non viable pour la traduction.

---

## v0.5 (2026-03-13) — Reconstruction complete de la table (regression)

### Modification
- Reconstruction de la table de chaines de zero au lieu de la relocation en fin
  de table (v0.4). Toutes les entrees sont empaquetees sequentiellement.

### Probleme
Le scan a 4 octets corrompait toujours les 22 entiers coincidant avec des offsets.
De plus, le fichier `NECEMEM.snx` non-standard etait corrompu lors de la
reconstruction, tronquant l'archive de 46 octets et empechant le jeu de demarrer.

---

## v0.4.1 (2026-03-13) — Correction ecriture en place

### Bugfix
- **Corrige la corruption sequentielle de la table de chaines** : la v0.4
  mettait a jour le prefixe `slen` avec la nouvelle taille (plus courte), ce qui
  desynchronisait le parseur sequentiel du moteur.

### Correction
- Conservation du `slen` original dans le prefixe de longueur. Le padding nul
  apres le texte est inoffensif car la chaine est terminee par `\x00`.

---

## v0.4 (2026-03-13) — Relocation en fin de table (debut du scan par pattern)

### Approche
- Ecriture en place pour les traductions plus courtes.
- Relocation en fin de table pour les traductions plus longues.
- Premier scan par pattern (`[0x11][0x02]`, `[0x0F]`, `[0x10]`, `[0x15]`)
  au lieu de mise a jour aveugle.

### Probleme
Le scan a pas de 4 octets confondait `[0x11][0x00][entier]` avec
`[0x11][0x02][offset]`, corrompant 22 valeurs entieres sur NV30.snx.
Crash du moteur a `lcsebody+0x11d21`.

---

## v0.3 (2026-03-10) — Mise a jour sans filtre (regression)

### Modification
- Mise a jour de TOUTE valeur correspondant a un ancien offset de chaine,
  sans filtre par opcode. Corrigeait les 21 references manquees de la v0.2
  mais introduisait des faux positifs massifs.

---

## v0.2 (2026-03-10) — Shift-JIS natif, patch mode, format simplifie

### Nouvelles fonctionnalites
- Commande `patch` pour remplacer uniquement les fichiers modifies.
- Sortie Shift-JIS native (plus de conversion UTF-8).
- Format 4 colonnes au lieu de 5.
- Auto-detection de la cle XOR depuis le `.lst`.

---

## v0.1 (2026-03-09) — Version initiale

### Fonctionnalites
- `unpack`, `pack`, `snx2txt`, `txt2snx`, modes batch.
- Reverse-engineering du format SNX et de l'archive LST.

### References techniques
- [GARbro](https://github.com/morkt/GARbro) — `ArcFormats/Tactics/ArcLST.cs`
- [LCSELocalizationTools](https://github.com/cqjjjzr/LCSELocalizationTools) — outil Java
- [LCScriptEngineTools](https://github.com/fengberd/LCScriptEngineTools) — script PHP
- [The MOON Kit](https://www.asceai.net/moonkit/) — documentation SNX (MOON.DVD)
- [lcsebody-main](https://github.com/) — decompileur Rust, documentation bytecode 12 octets
- Desassemblage de `lcsebody.exe` avec Capstone x86
