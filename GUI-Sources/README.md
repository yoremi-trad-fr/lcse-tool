# LCSE Tool GUI

Interface Wails/Svelte pour `lcse-tool`, pensée pour le workflow de traduction FR de
One Nexton.

## Utilisation

- Placer `lcse-tool.exe` à côté de `LCSEToolGUI.exe`, ou cliquer sur le chemin
  dans la barre du haut pour le localiser.
- Les outils wrapper sont fournis dans `bin/` :
  `lcse-tool.exe`, `moon_asm.exe`, `moon_extractTGF.exe`, `moon_patch.exe`.
- Le binaire compilé est généré dans `build/bin/LCSEToolGUI.exe`.
- Les opérations TSV remplacent les anciens scripts interactifs `Extract.py` et
  `Reinject.py` côté interface.
- L'onglet MOON permet d'extraire `moon_jp` / `moon_eng`, d'exporter les images
  TGF en PNG, de convertir les scripts assembleur en UTF-8, d'exporter/importer
  les lignes `SETSTATUS` et `TEXT`, puis d'assembler les SNX avec `moon_asm.exe`.

## Développement

```bash
wails dev
```

## Build

```bash
wails build
```
