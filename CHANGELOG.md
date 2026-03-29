# CHANGELOG

## v0.8 (2026-03-29) — Accents français fonctionnels

### Changement d'encodage des accents
- **Single-byte (0xA1-0xAD)** au lieu de double-byte SJIS (F040-F04C).
  Le moteur LCSE code en dur l'avance du curseur : 24px pour les double-byte,
  12px pour les single-byte. L'ancien encodage produisait un espace parasite
  après chaque accent. La plage half-width katakana (0xA1-0xAD) est utilisée
  car le moteur la traite comme single-byte (avance 12px).

### Hook GetGlyphOutlineA (remplace CreateFontIndirectA seul)
- **Découverte** : le moteur n'utilise ni `TextOutA` ni `ExtTextOutA`.
  Il appelle `GetGlyphOutlineA` (format `GGO_GRAY4_BITMAP`) pour obtenir le
  bitmap de chaque glyphe, puis le compose via `BitBlt`/`StretchBlt`.
  Identifié par analyse des imports PE de `lcsebody.exe`.
- **Hook `GetGlyphOutlineA`** : intercepte les bytes 0xA1-0xAD et appelle
  `GetGlyphOutlineW` avec le vrai codepoint Unicode (ex: 0xA1 → U+00E9 = é).
  Court-circuite entièrement le mapping SJIS → PUA.
- **Fix sign-extension** : le moteur passe les caractères via `char` signé.
  Le byte 0xA1 (-95) arrive comme `0xFFFFFFA1` après cast en `UINT`.
  Le hook masque avec `& 0xFF` pour détecter la plage correctement.

### Police
- **MS Gothic par défaut** : police Unicode complète, contient tous les accents
  français nativement. Pas besoin de police custom ni de glyphes PUA.
- Le hook `CreateFontIndirectA` est conservé pour la substitution de police
  (le moteur demande `ＭＳ ゴシック` fullwidth depuis INIT.snx entrées 12-15).

### Problèmes résolus (historique de debug)
1. **v0.7** : hook `CreateFontIndirectA` seul — police substituée mais accents
   affichés en "A" (le moteur n'utilise pas TextOutA).
2. **INI incorrect** : `Name=UD Digi Kyokasho N` au lieu de `N-R` — la police
   n'était jamais sélectionnée par Windows.
3. **Police sans PUA** : glyphes absents aux positions U+E000-U+E00C — patchée
   via fontTools, mais rendu toujours en "A" car `GetGlyphOutlineA` ne résolvait
   pas le mapping cp932 → PUA correctement.
4. **TextOutA/ExtTextOutA absents de l'IAT** : confirmé par le log v2
   (`FAILED to patch`). Le moteur utilise `GetGlyphOutlineA` exclusivement.
5. **Hook GetGlyphOutlineA (v3)** : accents enfin visibles, mais espacement
   double (24px) car les bytes F040-F04C sont double-byte SJIS.
6. **Passage single-byte (v0.8)** : bytes 0xA1-0xAD, avance correcte de 12px.
   Mais le hook ne matchait pas — le byte 0xA1 arrivait comme 0xFFFFFFA1.
7. **Fix sign-extension (v5.2)** : masque `& 0xFF`, accents fonctionnels.

---

## v0.7 (2026-03-15) — UTF-8 + Font Hook + Scripts extraction

### Nouvelles fonctionnalites
- **Export UTF-8 BOM** : `snx2txt` exporte desormais en UTF-8 avec BOM.
  Les fichiers s'ouvrent directement en UTF-8 dans Notepad++ avec le japonais
  lisible et editable.
- **Import UTF-8** : `txt2snx` detecte automatiquement l'encodage (UTF-8 BOM
  ou Shift-JIS) et convertit correctement vers SJIS pour le moteur.
- **Mapping accents** : les caracteres accentues (e, e, c, a, a, u, o, e, i,
  u, e, i, u) sont convertis en codes SJIS user-defined (F040-F04C) pour
  compatibilite avec une police custom. Les accents non-supportes sont
  remplaces par leur lettre de base.
- **Scripts Python** : `Extract.py` et `Reinject.py` pour extraire les lignes
  de dialogue et les reinjecter apres traduction, avec detection automatique
  de l'encodage.
- **Font Hook (DLL)** : `lcse_hook.dll` + `lcse_launcher.exe` pour charger
  une police custom (`lcse_font.ttf`) au lieu de MS Gothic. Permet un
  espacement proportionnel des lettres latines.

### Workflow complet
```
lcse-tool unpack lcsebody1 extracted/      # Extraire l'archive
lcse-tool snx2txt extracted/ scripts/      # SNX -> TXT (UTF-8 BOM)
python Extract.py                          # Extraire les dialogues -> TSV
  (traduire le TSV)
python Reinject.py                         # Reinjecter les dialogues
lcse-tool txt2snx-batch scripts/ extracted/ patched/  # TXT -> SNX
lcse-tool patch lcsebody1 patched/ lcsebody1_fr       # Patcher l'archive
```

---

## v0.6.1 (2026-03-14) — Protection fichiers non-standard + copie intacte

### Bugfix critique
- **Corrige le crash au lancement** cause par la corruption de `NECEMEM.snx`,
  un fichier SNX non-standard qui ne suit pas le format `8 + h0*12 + h1 = taille`.
- **Detection des SNX non-standard** : copie byte-for-byte sans modification.
- **Copie intacte si 0 modifications** : elimine tout risque de corruption.

---

## v0.6 (2026-03-14) — Scan 12-octets aligne (zero faux positifs)

### Correction fondamentale
- **Supprime la limite de taille des traductions.** La table de chaines est
  entierement reconstruite et les references mises a jour de maniere fiable.
- **Scan a pas de 12 octets** (taille d'une instruction LCSE).
  Seules les instructions `[0x11][0x02][offset]` sont mises a jour.
- Verification : **1063 refs = 1063 entrees, 0 faux positifs**.
- Source : reverse-engineering du bytecode LCSE via decompileur Rust
  (`bytecode notes.txt` : instructions 12 octets, opcode 0x11 types).

---

## v0.5.1 (2026-03-13) — Ecriture en place (approche temporaire)

- Ecriture strictement en place, sans modification du bytecode.
- Troncature des traductions trop longues. Abandonne en v0.6.

---

## v0.5 (2026-03-13) — Reconstruction complete (regression)

- Reconstruction de la table de zero. Crash du a NECEMEM + faux positifs.

---

## v0.4.1 (2026-03-13) — Correction slen

- Conservation du slen original dans le prefixe de longueur.

---

## v0.4 (2026-03-13) — Relocation en fin de table

- Premier scan par pattern (`[0x11][0x02]`).
- Crash du au scan a 4 octets (faux positifs sur entiers).

---

## v0.3 (2026-03-10) — Mise a jour sans filtre

- Mise a jour de toute valeur correspondant a un offset. Faux positifs massifs.

---

## v0.2 (2026-03-10) — Shift-JIS natif, patch mode

- Commande `patch`, sortie Shift-JIS, format 4 colonnes, auto-detection cle XOR.

---

## v0.1 (2026-03-09) — Version initiale

- `unpack`, `pack`, `snx2txt`, `txt2snx`, modes batch.
- Reverse-engineering du format SNX et de l'archive LST.

---

## References techniques

- [GARbro](https://github.com/morkt/GARbro) — format LST/Nexton
- [LCSELocalizationTools](https://github.com/cqjjjzr/LCSELocalizationTools) — outil Java
- [LCScriptEngineTools](https://github.com/fengberd/LCScriptEngineTools) — script PHP
- [The MOON Kit](https://www.asceai.net/moonkit/) — documentation SNX
- lcsebody-main (decompileur Rust) — documentation bytecode 12 octets
- Desassemblage de `lcsebody.exe` avec Capstone x86
