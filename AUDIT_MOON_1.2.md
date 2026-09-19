# Audit MOON / LCSE Tool 1.2

Date : 2026-09-19

## Périmètre

- Archives locales `moon_jp` et `moon_eng`, jamais modifiées.
- Version 1.1 de LCSE Tool et sources locales du dépôt.
- MOON Kit officiel utilisé uniquement pour `moon_asm.exe` et comme référence.

## Diagnostic de la 1.1

- L'extraction des deux archives fonctionnait déjà.
- Le bouton MOON SNX vers TXT ne désassemblait pas le SNX choisi : il copiait
  le fichier source japonais de même nom depuis `moon_scripts`.
- Le dossier `moon_scripts` annoncé n'était pas livré dans le dossier 1.1.
- La conversion batch ignorait les erreurs individuelles, ce qui pouvait
  afficher un résultat apparemment réussi avec des fichiers manquants.
- Ce mécanisme ne pouvait donc pas extraire le texte anglais réellement présent
  dans `moon_eng`.

## Correctifs 1.2

- Désassembleur dédié au bytecode variable de MOON, avec labels, expressions,
  commandes, chaînes Shift-JIS, `TEXT`, `SELECT` et `SETSTATUS`.
- Détection automatique, fichier par fichier, des SNX encore chiffrés par XOR
  `0xAA` dans l'archive anglaise.
- Gestion du fragment orphelin de 93 octets placé après `EOF` dans `INIT.snx`
  japonais, ignoré comme dans le MOON Kit officiel.
- Sortie TXT UTF-8 BOM directement depuis les SNX anglais ou japonais.
- Export/import des seules lignes traduisibles vers des fichiers `.dlg.txt`.
- Détection des six scripts japonais non scindés conservés comme résidus dans
  l'archive anglaise ; ils restent auditables mais ne sont pas réassemblés
  lorsque leurs remplaçants A/B sont présents.
- Encodage des accents français après passage dans `moon_asm.exe`, uniquement
  à l'intérieur des chaînes, sans risque de modifier les opcodes.
- Protection des caractères japonais dont le second octet Shift-JIS vaut
  `0x5C`, que l'assembleur confond sinon avec un antislash d'échappement.
- Lanceur `moon_launcher.exe` et installation du hook depuis la GUI.
- Remontée effective des erreurs de conversion batch.

## Résultats vérifiés

| Étape | Résultat |
|---|---:|
| Extraction `moon_jp` | 674 fichiers : 119 SNX, 555 TGF |
| Extraction `moon_eng` | 686 fichiers : 131 SNX, 479 TGF, 76 BMP |
| Désassemblage japonais | 119/119 SNX convertis |
| Désassemblage anglais | 131/131 SNX convertis |
| Dialogues japonais | 116 fichiers, 26 235 lignes, réimport identique |
| Scripts anglais actifs | 125 |
| Export des dialogues | 122 fichiers, 26 244 lignes |
| Réimport sans traduction | 26 244/26 244 lignes, identiques |
| Assemblage avec `moon_asm.exe` | 125/125 SNX |
| Assemblage japonais avec `moon_asm.exe` | 119/119 SNX, texte et commandes conservés |
| Redésassemblage de contrôle | 125/125 SNX valides, texte et commandes conservés |
| Rebuild de `moon_eng` | 125 fichiers remplacés sur 686 |
| Réextraction du rebuild | 686 fichiers, mêmes 131/479/76 types |
| Comparaison du rebuild | 125 patches exacts + 561 fichiers inchangés, 0 différence inattendue |

Les coupures de ligne internes sont une mise en page automatique : elles
peuvent se déplacer lors d'un nouvel assemblage, sans perte du texte.
Le plus gros SNX japonais reconstruit atteint 64 781 octets, très près de la
limite de 65 536 ; l'archive anglaise déjà scindée reste donc la cible de
rebuild la plus sûre pour une traduction française longue, même si le texte de
départ est choisi dans les fichiers japonais.

## Empreintes des entrées

- `moon_jp` : `38FFC531576E1F0C085DD519E51C7CC4B34BAA75BF31B019A218368E416057E2`
- `moon_jp.lst` : `78F99608254EE2C63B1F2C2EB4B6229F236F50105632E70D7DCD3DCEC955F3ED`
- `moon_eng` : `8117DCFAC4CCB84940F22C1882EE2403040773A1AFAA162399A119C321B88814`
- `moon_eng.lst` : `8F70F66CA596D5E5C8B7E41A5586585E430685002F883FA640C9032E592C9AC8`
- MOON Kit téléchargé : `B02E4AAD7364CB392712D3D6A196DE310D2B6563CFC6D08125F3134859E28EC0`

## Point restant à valider dans le jeu

Le pipeline produit bien les octets d'accents `0xA1-0xAD` attendus et
`MOON_eng.EXE` importe les deux fonctions GDI interceptées par le hook. Le rendu
visuel d'une phrase française dans une scène réelle de MOON n'a toutefois pas
encore été validé à l'écran. Il faudra faire ce test sur une copie jouable avant
de considérer la partie graphique comme définitivement close.

Référence : <https://asceai.net/moonkit/>
