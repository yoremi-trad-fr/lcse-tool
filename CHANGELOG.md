# CHANGELOG

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
