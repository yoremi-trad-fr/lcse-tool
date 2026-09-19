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
- L'onglet **Hook accents** installe le lanceur et la DLL pour ONE ou
  `MOON_eng.EXE`.

## Développement

```bash
wails dev
```

## Build

```bash
wails build
```
