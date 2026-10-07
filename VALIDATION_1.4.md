# Validation de la version 1.4 — 7 octobre 2026

- GUI Windows x64 : 1.4.0 ; CLI : 1.4 ; hook Windows x86 : 1.4.0.
- Tests Go locaux GUI, installation MOON avec moteur réel et `go vet` : réussis.
- Compilation de production Wails/Svelte : sans obfuscation ni compression.

## Images

- Les 479 TGF de l'archive anglaise produisent les mêmes octets BMP que
  l'extracteur officiel, y compris l'image monochrome FG165.
- Deux PNG (120 × 72 et 160 × 100) convertis en TGF puis relus par l'extracteur
  officiel conservent tous leurs pixels. Les sources restent intactes.
- Tests des alignements BMP, séquences longues, queues de 1/2 octets, flux
  tronqués, tailles excessives, alpha partiel et sorties déjà existantes.

## Lancement MOON et accents

- Le `MOON_FR.bat` distribué démarre un pilote GDI de contrôle via le dossier
  `locale/` et le profil `JAP` réels du jeu (Locale Emulator 2.5.0.1).
- Ce pilote confirme la page de codes 932, la locale 0x0411, son dossier
  courant et l'import normal de `GetGlyphOutlineA` depuis `lcse_hook.dll`.
- Les 13 accents rendent les mêmes métriques et pixels que les appels Unicode
  GDI pour 0xA1–0xAD et 0xFFFFFFA1–0xFFFFFFAD (26 cas).
- ASCII et Shift-JIS sont conservés. Neuf exports GDI transmis à la bibliothèque
  système sont vérifiés par leur adresse.
- Sur le moteur MOON réel, la section `.text`, le point d'entrée et les
  adresses de ses imports restent identiques après préparation.
- Sauvegarde originale vérifiée, réinstallation sans double modification,
  suppression des anciens lanceurs après sauvegarde et refus d'une sauvegarde
  altérée ou hors du dossier du jeu : testés.
- Les archives, dialogues et banques audio du jeu ne sont pas reconstruits.

Le pilote teste le lancement et le rendu GDI ; il ne valide pas une partie du
jeu complet. La progression et le rendu des dialogues restent à confirmer en jeu.

## Antivirus et livraison

Les fichiers sont compilés sans obfuscation ni compression. Le hook utilise
les imports Windows normaux ; sa configuration est chargée hors de `DllMain`.
Le lanceur MOON utilise le dossier Locale Emulator déjà fourni avec le jeu.
Aucune exclusion antivirus n'est nécessaire à l'installation. Defender,
signatures 1.459.576.0 et protection temps réel active, ne détecte aucune menace
dans le paquet Windows corrigé ni dans le moteur MOON installé lors des scans
locaux du 7 octobre 2026. Aucune exclusion n'a été ajoutée.

Les résultats d'analyse locale et les empreintes du paquet sont fournis avec
les fichiers de livraison. Une analyse locale ne prédit pas les détections avec
des signatures futures. Les binaires distribués ne sont pas signés.
