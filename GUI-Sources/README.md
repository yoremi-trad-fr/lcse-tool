# LCSE Tool GUI

Interface Wails/Svelte pour `lcse-tool`, pensée pour les workflows de traduction
française de ONE et MOON.

## Utilisation

- Placer `lcse-tool.exe` à côté de `LCSEToolGUI.exe`, ou cliquer sur le chemin
  dans la barre du haut pour le localiser.
- Les outils wrapper sont fournis dans `bin/` :
  `lcse-tool.exe`, `moon_asm.exe`, `moon_extractTGF.exe`, `moon_patch.exe`.
- Les anciens SNX de MOON sont désassemblés directement en UTF-8 par
  `lcse-tool.exe`. Seul `moon_asm.exe`, provenant du
  [MOON Kit](https://asceai.net/moonkit/), reste nécessaire au rebuild.
- Le binaire compilé est généré dans `build/bin/LCSEToolGUI.exe`.
- Les opérations TSV remplacent les anciens scripts interactifs `Extract.py` et
  `Reinject.py` côté interface.
- L'onglet MOON permet d'extraire `moon_jp` / `moon_eng`, d'exporter les images
  TGF en PNG, de convertir les scripts assembleur en UTF-8, d'exporter/importer
  les lignes `SETSTATUS` et `TEXT`, puis d'assembler les SNX avec `moon_asm.exe`.
  Les six anciens scripts japonais non scindés de l'archive anglaise sont
  conservés pour l'audit mais exclus automatiquement du rebuild.
- L'écran **Images MOON -> PNG** accepte une archive MOON avec son `.lst`, un
  dossier ou un fichier TGF/BMP. Les TGF compressés passent par
  `moon_extractTGF.exe` ; les BMP bruts sont convertis directement. Les PNG
  sont écrits dans le dossier choisi, sans modifier les images source.
- Pour **Rebuild archive**, choisir un autre chemin de sortie que l'archive
  source. Le patch ne remplace que les fichiers de même nom et extension ; un
  PNG ne remplace pas une entrée BMP. Les fichiers ignorés sont signalés.
- L'onglet **Hook accents** installe le lanceur et la DLL pour ONE. Pour MOON,
  il crée `MOON_fr.exe`, qui lance le véritable `MOON_eng.EXE` sans le
  renommer afin que le moteur continue de charger l'archive `moon_eng`.

## Développement

```bash
wails dev
```

## Build

```bash
wails build
```
