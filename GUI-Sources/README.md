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
- L'écran **PNG<->TGF** propose **PNG -> TGF** et **TGF -> PNG**.
  Choisir un ou plusieurs fichiers, ou un dossier ; l'archive avec son `.lst`
  n'est proposée qu'en TGF -> PNG et seuls les TGF sont convertis. Les BMP
  sont déjà lisibles après extraction. Choisir un dossier de sortie vide :
  les sources et les fichiers existants sont conservés. Le codec est natif.
  Les pixels entièrement transparents deviennent magenta (couleur-clé du jeu).
  La transparence partielle est refusée.
- Pour **Rebuild archive**, choisir un autre chemin de sortie que l'archive
  source. Le patch ne remplace que les fichiers de même nom et extension ; un
  PNG ne remplace pas une entrée BMP. Les fichiers ignorés sont signalés.
- L'onglet **Hook accents** sauvegarde l'original puis prépare le moteur
  pour charger les accents normalement. Pour MOON, il conserve `MOON_eng.EXE`
  à côté des archives et installe un seul lanceur : `MOON_FR.bat`, qui utilise
  le profil japonais `JAP` du dossier `locale/` du jeu. La configuration
  est `lcse_hook.ini` dans le dossier du jeu. Les anciens lanceurs sont
  sauvegardés puis retirés. Pour ONE, la copie reste dans `lcse_fr/` ; lancer
  `lcsebody_fr.exe` et régler `lcse_fr/lcse_hook.ini`.

## Développement

```bash
wails dev
```

## Build

```bash
wails build
```
