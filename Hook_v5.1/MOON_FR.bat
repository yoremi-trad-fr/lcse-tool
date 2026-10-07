@echo off
setlocal
cd /d "%~dp0"
title MOON. - Version francaise

if not exist "MOON_eng.EXE" goto missing_engine
if not exist "moon_eng" goto missing_archive
if not exist "moon_eng.lst" goto missing_archive
if not exist "ps0" goto missing_audio
if not exist "ps0.lst" goto missing_audio
if not exist "ps1" goto missing_voices
if not exist "ps1.lst" goto missing_voices
if not exist "locale\LEProc.exe" goto missing_locale
if not exist "lcse_hook.dll" goto missing_hook
if not exist "lcse_hook_install.json" goto missing_hook

pushd "%~dp0locale"
"%~dp0locale\LEProc.exe" -runas JAP "%~dp0MOON_eng.EXE"
set "moon_result=%errorlevel%"
popd
if not "%moon_result%"=="0" goto launch_error
exit /b 0

:missing_engine
echo MOON_eng.EXE est introuvable dans le dossier du jeu.
goto failed
:missing_archive
echo L'archive moon_eng et son index moon_eng.lst sont requis.
goto failed
:missing_audio
echo La banque sonore ps0 et son index ps0.lst sont requis.
goto failed
:missing_voices
echo La banque de voix ps1 et son index ps1.lst sont requis.
echo ps1.exe est une archive compressee et ne remplace pas ps1.
goto failed
:missing_locale
echo Le dossier locale du jeu est incomplet : LEProc.exe est requis.
goto failed
:missing_hook
echo Installer le hook MOON depuis LCSE Tool GUI 1.4 dans ce dossier.
goto failed
:launch_error
echo Locale Emulator a signale une erreur de lancement : %moon_result%.
:failed
pause
exit /b 1
