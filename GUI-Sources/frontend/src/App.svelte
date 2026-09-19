<script>
  import { onMount, onDestroy, tick } from 'svelte';
  import logo from './assets/images/logo-universal.png';
  import { EventsOn, EventsOff } from '../wailsjs/runtime/runtime.js';
  import {
    GetLCSEPath,
    GetToolPaths,
    SetLCSEPath,
    SelectArchiveBase,
    SelectSNXFile,
    SelectTXTFile,
    SelectAnyFile,
    SelectDirectory,
    SelectSaveFile,
    StopProcess,
    UnpackArchive,
    SNXToTXT,
    TXTToSNX,
    TXTToSNXBatch,
    PatchArchive,
    PackArchive,
    MoonUnpackArchive,
    MoonExtractImagesFromArchive,
    MoonConvertImages,
    MoonSNXToTXT,
    MoonTXTToSNX,
    MoonAssembleScripts,
    MoonExportDialogues,
    MoonImportDialogues,
    OneExportDialogues,
    OneImportDialogues,
    GetOneHookConfig,
    SaveOneHookConfig,
    InstallOneHook,
    InstallMoonHook
  } from '../wailsjs/go/main/App.js';

  let selectedOp = 'prepare';
  let running = false;
  let lcsePath = '';
  let toolPaths = {};
  let consoleLines = [];
  let consoleEl;

  let prepGame = 'one';
  let prepArchive = '';
  let prepExtractDir = '';
  let prepTxtDir = '';
  let prepKey = '';
  let prepSnxKey = '';

  let dialogueGame = 'one';
  let dialogueScripts = '';
  let dialogueOutput = '';
  let importScripts = '';
  let importDialogues = '';
  let importOutput = '';

  let snxGame = 'one';
  let snxDirection = 's2t';
  let snxBatch = true;
  let snxInput = '';
  let snxOriginal = '';
  let snxOutput = '';

  let imageArchive = '';
  let imageArchiveOutput = '';
  let imageInput = '';
  let imageOutput = '';

  let rebuildGame = 'one';
  let rebuildArchive = '';
  let rebuildPatchDir = '';
  let rebuildOutput = '';
  let rebuildKey = '';
  let rebuildSnxKey = '';
  let packInputDir = '';
  let packOutput = '';
  let packKey = '';
  let packSnxKey = '';

  let hookDir = '';
  let hookFontName = 'MS Gothic';
  let hookDebugLog = '0';
  let hookGameDir = '';
  let hookGame = 'one';

  const operations = [
    { id: '_flow', label: 'Workflow', section: true },
    { id: 'prepare', label: 'Préparation' },
    { id: 'dialogues', label: 'Import/export dialogues' },
    { id: '_tools', label: 'Outils', section: true },
    { id: 'snx', label: 'SNX <-> TXT' },
    { id: 'images', label: 'TGF <-> PNG' },
    { id: 'rebuild', label: 'Rebuild archive' },
    { id: 'hook', label: 'Hook accents' },
    { id: '_info', label: '', section: true },
    { id: 'about', label: 'À propos' }
  ];

  let pendingLines = [];
  let flushTimer = null;

  function addLine(text) {
    let cls = '';
    if (text.includes('[OK]')) cls = 'line-ok';
    else if (text.includes('[ERROR]') || text.includes('ERROR')) cls = 'line-err';
    else if (text.includes('[STOP]')) cls = 'line-warn';
    else if (text.startsWith('>')) cls = 'line-cmd';
    else if (text.startsWith('=')) cls = 'line-sep';
    pendingLines.push({ text, cls });
    if (!flushTimer) flushTimer = setTimeout(flushConsole, 70);
  }

  function flushConsole() {
    if (pendingLines.length > 0) {
      consoleLines = [...consoleLines, ...pendingLines];
      pendingLines = [];
      if (consoleLines.length > 2000) consoleLines = consoleLines.slice(-1500);
      tick().then(() => {
        if (consoleEl) consoleEl.scrollTop = consoleEl.scrollHeight;
      });
    }
    flushTimer = null;
  }

  function clearConsole() {
    consoleLines = [];
    pendingLines = [];
  }

  async function run(task) {
    if (running) return;
    running = true;
    try {
      await task();
    } catch (err) {
      addLine('[ERROR] ' + err);
    } finally {
      running = false;
    }
  }

  onMount(async () => {
    EventsOn('log', (msg) => addLine(msg));
    lcsePath = await GetLCSEPath();
    toolPaths = await GetToolPaths();
    const hook = await GetOneHookConfig();
    hookDir = hook.hookDir || '';
    hookFontName = hook.fontName || 'MS Gothic';
    hookDebugLog = hook.debugLog || '0';

    addLine('LCSE Tool GUI - Nexton');
    addLine(lcsePath ? 'lcse-tool: ' + lcsePath : '[ERROR] lcse-tool introuvable');
    if (toolPaths.moonAsm) addLine('moon_asm: ' + toolPaths.moonAsm);
    if (toolPaths.moonTGF) addLine('moon_extractTGF: ' + toolPaths.moonTGF);
    if (toolPaths.moonScripts) addLine('sources MOON: ' + toolPaths.moonScripts);
    if (hookDir) addLine('hook accents: ' + hookDir);
    addLine('Prêt.');
  });

  onDestroy(() => {
    EventsOff('log');
  });

  async function locateLCSE() {
    lcsePath = await SetLCSEPath();
    if (lcsePath) addLine('lcse-tool: ' + lcsePath);
  }

  async function stopProcess() {
    await StopProcess();
  }

  function selectOp(op) {
    if (!op.section) selectedOp = op.id;
  }

  async function pickArchive(setter) {
    const value = await SelectArchiveBase();
    if (value) setter(value);
  }

  async function pickSNX(setter) {
    const value = await SelectSNXFile();
    if (value) setter(value);
  }

  async function pickTXT(setter) {
    const value = await SelectTXTFile();
    if (value) setter(value);
  }

  async function pickAny(title, setter) {
    const value = await SelectAnyFile(title);
    if (value) setter(value);
  }

  async function pickDir(title, setter) {
    const value = await SelectDirectory(title);
    if (value) setter(value);
  }

  async function saveFile(title, defaultName, pattern, desc, setter) {
    const value = await SelectSaveFile(title, defaultName, pattern, desc);
    if (value) setter(value);
  }

  function chooseSNXInput() {
    if (snxDirection === 's2t') {
      if (snxBatch) return pickDir('Dossier SNX', (v) => snxInput = v);
      return pickSNX((v) => snxInput = v);
    }
    if (snxBatch) return pickDir('Dossier TXT', (v) => snxInput = v);
    return pickTXT((v) => snxInput = v);
  }

  function chooseSNXOriginal() {
    if (snxBatch) return pickDir('Dossier SNX originaux', (v) => snxOriginal = v);
    return pickSNX((v) => snxOriginal = v);
  }

  function chooseSNXOutput() {
    if (snxBatch) return pickDir('Dossier sortie', (v) => snxOutput = v);
    if (snxDirection === 's2t') {
      return saveFile('TXT de sortie', 'script.txt', '*.txt', 'Fichiers texte', (v) => snxOutput = v);
    }
    return saveFile('SNX de sortie', 'script.snx', '*.snx', 'Fichiers SNX', (v) => snxOutput = v);
  }

  function startPrepare() {
    run(async () => {
      if (prepGame === 'one') {
        const unpacked = await UnpackArchive(prepArchive, prepExtractDir, prepKey, prepSnxKey);
        if (unpacked === 'OK' && prepTxtDir) await SNXToTXT(prepExtractDir, prepTxtDir);
        return;
      }
      const unpacked = await MoonUnpackArchive(prepArchive, prepExtractDir);
      if (unpacked === 'OK' && prepTxtDir) await MoonSNXToTXT(prepExtractDir, prepTxtDir);
    });
  }

  function startExtractOnly() {
    if (prepGame === 'one') run(() => UnpackArchive(prepArchive, prepExtractDir, prepKey, prepSnxKey));
    else run(() => MoonUnpackArchive(prepArchive, prepExtractDir));
  }

  function startDialogueExport() {
    if (dialogueGame === 'one') run(() => OneExportDialogues(dialogueScripts, dialogueOutput));
    else run(() => MoonExportDialogues(dialogueScripts, dialogueOutput));
  }

  function startDialogueImport() {
    if (dialogueGame === 'one') run(() => OneImportDialogues(importScripts, importDialogues, importOutput));
    else run(() => MoonImportDialogues(importScripts, importDialogues, importOutput));
  }

  function startSNXConvert() {
    if (snxGame === 'one' && snxDirection === 's2t') {
      run(() => SNXToTXT(snxInput, snxOutput));
      return;
    }
    if (snxGame === 'one' && snxDirection === 't2s') {
      if (snxBatch) run(() => TXTToSNXBatch(snxInput, snxOriginal, snxOutput));
      else run(() => TXTToSNX(snxInput, snxOriginal, snxOutput));
      return;
    }
    if (snxGame === 'moon' && snxDirection === 's2t') {
      run(() => MoonSNXToTXT(snxInput, snxOutput));
      return;
    }
    if (snxBatch) run(() => MoonAssembleScripts(snxInput, snxOutput));
    else run(() => MoonTXTToSNX(snxInput, snxOutput));
  }

  function startMoonArchiveImages() {
    run(() => MoonExtractImagesFromArchive(imageArchive, imageArchiveOutput));
  }

  function startMoonDirectImages() {
    run(() => MoonConvertImages(imageInput, imageOutput));
  }

  function startArchiveRebuild() {
    if (rebuildGame === 'one') {
      run(() => PatchArchive(rebuildArchive, rebuildPatchDir, rebuildOutput, rebuildKey, rebuildSnxKey));
    } else {
      run(() => PatchArchive(rebuildArchive, rebuildPatchDir, rebuildOutput, '', ''));
    }
  }

  function startPackArchive() {
    run(() => PackArchive(packInputDir, packOutput, packKey, packSnxKey));
  }

  async function reloadHookConfig() {
    const hook = await GetOneHookConfig();
    hookDir = hook.hookDir || '';
    hookFontName = hook.fontName || 'MS Gothic';
    hookDebugLog = hook.debugLog || '0';
    if (hookDir) addLine('hook ONE: ' + hookDir);
  }

  function saveHookConfig() {
    run(() => SaveOneHookConfig(hookFontName, hookDebugLog));
  }

  function installHook() {
    if (hookGame === 'moon') run(() => InstallMoonHook(hookGameDir, hookFontName, hookDebugLog));
    else run(() => InstallOneHook(hookGameDir, hookFontName, hookDebugLog));
  }
</script>

<div id="app">
  <header class="titlebar">
    <div class="title-main">
      <img src={logo} alt="" />
      <span>LCSE Tool GUI</span>
    </div>
    <button class="path-button" on:click={locateLCSE} title="Changer lcse-tool">
      {lcsePath || 'lcse-tool introuvable'}
    </button>
  </header>

  <div class="content">
    <aside class="sidebar">
      <div class="sidebar-title">Options</div>
      <div class="sidebar-list">
        {#each operations as op}
          {#if op.section}
            <div class="sidebar-section">{op.label}</div>
          {:else}
            <button class:active={selectedOp === op.id} on:click={() => selectOp(op)}>{op.label}</button>
          {/if}
        {/each}
      </div>
    </aside>

    <main class="workspace">
      {#if selectedOp === 'prepare'}
        <section class="form-view">
          <h1>Préparation</h1>
          <div class="block full">
            <div class="form-grid">
              <label>Jeu</label>
              <select bind:value={prepGame}>
                <option value="one">ONE</option>
                <option value="moon">MOON</option>
              </select>

              <label>Archive</label>
              <div class="row"><input bind:value={prepArchive} readonly /><button on:click={() => pickArchive((v) => prepArchive = v)}>Parcourir</button></div>

              <label>Dossier extrait</label>
              <div class="row"><input bind:value={prepExtractDir} readonly /><button on:click={() => pickDir('Dossier extrait', (v) => prepExtractDir = v)}>Parcourir</button></div>

              <label>Dossier TXT</label>
              <div class="row"><input bind:value={prepTxtDir} readonly /><button on:click={() => pickDir('Dossier TXT', (v) => prepTxtDir = v)}>Parcourir</button></div>

              {#if prepGame === 'one'}
                <label>Clés</label>
                <div class="row compact"><input bind:value={prepKey} placeholder="LST auto" /><input bind:value={prepSnxKey} placeholder="SNX auto" /></div>
              {/if}
            </div>
            <div class="actions">
              <button class="primary" on:click={startPrepare} disabled={running || !prepArchive || !prepExtractDir}>Préparer</button>
              <button on:click={startExtractOnly} disabled={running || !prepArchive || !prepExtractDir}>Extraire</button>
            </div>
          </div>
        </section>

      {:else if selectedOp === 'dialogues'}
        <section class="form-view">
          <h1>Import/export dialogues</h1>
          <div class="toolbar-line">
            <label>Jeu</label>
            <select bind:value={dialogueGame}>
              <option value="one">ONE</option>
              <option value="moon">MOON</option>
            </select>
          </div>

          <div class="split">
            <div class="block">
              <h2>Exporter</h2>
              <div class="form-grid">
                <label>Scripts TXT</label>
                <div class="row"><input bind:value={dialogueScripts} readonly /><button on:click={() => pickDir('Dossier scripts TXT', (v) => dialogueScripts = v)}>Parcourir</button></div>
                <label>Dialogues</label>
                <div class="row"><input bind:value={dialogueOutput} readonly /><button on:click={() => pickDir('Dossier dialogues', (v) => dialogueOutput = v)}>Parcourir</button></div>
              </div>
              <div class="actions left">
                <button class="primary" on:click={startDialogueExport} disabled={running || !dialogueScripts || !dialogueOutput}>Exporter</button>
              </div>
            </div>

            <div class="block">
              <h2>Importer</h2>
              <div class="form-grid">
                <label>Scripts base</label>
                <div class="row"><input bind:value={importScripts} readonly /><button on:click={() => pickDir('Dossier scripts base', (v) => importScripts = v)}>Parcourir</button></div>
                <label>Dialogues</label>
                <div class="row"><input bind:value={importDialogues} readonly /><button on:click={() => pickDir('Dossier dialogues', (v) => importDialogues = v)}>Parcourir</button></div>
                <label>Scripts sortie</label>
                <div class="row"><input bind:value={importOutput} readonly /><button on:click={() => pickDir('Dossier scripts sortie', (v) => importOutput = v)}>Parcourir</button></div>
              </div>
              <div class="actions left">
                <button class="primary" on:click={startDialogueImport} disabled={running || !importScripts || !importDialogues || !importOutput}>Importer</button>
              </div>
            </div>
          </div>
        </section>

      {:else if selectedOp === 'snx'}
        <section class="form-view">
          <h1>SNX &lt;-&gt; TXT</h1>
          <div class="toolbar-line">
            <label>Jeu</label>
            <select bind:value={snxGame}>
              <option value="one">ONE</option>
              <option value="moon">MOON</option>
            </select>
            <label>Sens</label>
            <select bind:value={snxDirection}>
              <option value="s2t">SNX -> TXT</option>
              <option value="t2s">TXT -> SNX</option>
            </select>
            <label class="check inline"><input type="checkbox" bind:checked={snxBatch} /> Batch</label>
          </div>

          {#if snxGame === 'moon' && snxDirection === 's2t'}
            <p class="notice">
              Les anciens SNX de MOON sont désassemblés directement en scripts UTF-8. L’archive anglaise mixte est prise en
              charge, y compris ses fichiers japonais résiduels chiffrés ; les six anciens scripts non scindés sont exclus lors du rebuild.
            </p>
          {/if}

          <div class="form-grid">
            <label>{snxDirection === 's2t' ? (snxBatch ? 'Dossier SNX' : 'Fichier SNX') : (snxBatch ? 'Dossier TXT' : 'Fichier TXT')}</label>
            <div class="row"><input bind:value={snxInput} readonly /><button on:click={chooseSNXInput}>Parcourir</button></div>

            {#if snxGame === 'one' && snxDirection === 't2s'}
              <label>{snxBatch ? 'SNX originaux' : 'SNX original'}</label>
              <div class="row"><input bind:value={snxOriginal} readonly /><button on:click={chooseSNXOriginal}>Parcourir</button></div>
            {/if}

            <label>{snxBatch ? 'Dossier sortie' : (snxDirection === 's2t' ? 'TXT sortie' : 'SNX sortie')}</label>
            <div class="row"><input bind:value={snxOutput} readonly /><button on:click={chooseSNXOutput}>Parcourir</button></div>
          </div>

          <div class="actions">
            <button class="primary" on:click={startSNXConvert} disabled={running || !snxInput || (snxGame === 'one' && snxDirection === 't2s' && !snxOriginal)}>
              Convertir
            </button>
          </div>
        </section>

      {:else if selectedOp === 'images'}
        <section class="form-view">
          <h1>TGF &lt;-&gt; PNG</h1>
          <div class="split">
            <div class="block">
              <h2>Archive MOON -> PNG</h2>
              <div class="form-grid">
                <label>Archive</label>
                <div class="row"><input bind:value={imageArchive} readonly /><button on:click={() => pickArchive((v) => imageArchive = v)}>Parcourir</button></div>
                <label>Dossier PNG</label>
                <div class="row"><input bind:value={imageArchiveOutput} readonly /><button on:click={() => pickDir('Dossier PNG', (v) => imageArchiveOutput = v)}>Parcourir</button></div>
              </div>
              <div class="actions left">
                <button class="primary" on:click={startMoonArchiveImages} disabled={running || !imageArchive || !imageArchiveOutput}>Extraire PNG</button>
              </div>
            </div>

            <div class="block">
              <h2>TGF/BMP -> PNG</h2>
              <div class="form-grid">
                <label>Source</label>
                <div class="row"><input bind:value={imageInput} readonly /><button on:click={() => pickAny('Source TGF/BMP', (v) => imageInput = v)}>Parcourir</button></div>
                <label>Dossier PNG</label>
                <div class="row"><input bind:value={imageOutput} readonly /><button on:click={() => pickDir('Dossier PNG', (v) => imageOutput = v)}>Parcourir</button></div>
              </div>
              <div class="actions left">
                <button class="primary" on:click={startMoonDirectImages} disabled={running || !imageInput || !imageOutput}>Convertir PNG</button>
              </div>
            </div>
          </div>
        </section>

      {:else if selectedOp === 'rebuild'}
        <section class="form-view">
          <h1>Rebuild archive</h1>
          <div class="toolbar-line">
            <label>Jeu</label>
            <select bind:value={rebuildGame}>
              <option value="one">ONE</option>
              <option value="moon">MOON</option>
            </select>
          </div>

          <div class="split">
            <div class="block">
              <h2>Patcher archive</h2>
              <div class="form-grid">
                <label>Archive</label>
                <div class="row"><input bind:value={rebuildArchive} readonly /><button on:click={() => pickArchive((v) => rebuildArchive = v)}>Parcourir</button></div>
                <label>Patch</label>
                <div class="row"><input bind:value={rebuildPatchDir} readonly /><button on:click={() => pickDir('Dossier patch', (v) => rebuildPatchDir = v)}>Parcourir</button></div>
                <label>Sortie</label>
                <div class="row"><input bind:value={rebuildOutput} /><button on:click={() => saveFile('Archive de sortie', rebuildGame === 'one' ? 'lcsebody_custom' : 'moon_custom', '*.*', 'Tous les fichiers', (v) => rebuildOutput = v)}>Parcourir</button></div>
                {#if rebuildGame === 'one'}
                  <label>Clés</label>
                  <div class="row compact"><input bind:value={rebuildKey} placeholder="LST auto" /><input bind:value={rebuildSnxKey} placeholder="SNX auto" /></div>
                {/if}
              </div>
              <div class="actions left">
                <button class="primary" on:click={startArchiveRebuild} disabled={running || !rebuildArchive || !rebuildPatchDir || !rebuildOutput}>Rebuild</button>
              </div>
            </div>

            <div class="block">
              <h2>Pack dossier</h2>
              <div class="form-grid">
                <label>Dossier</label>
                <div class="row"><input bind:value={packInputDir} readonly /><button on:click={() => pickDir('Dossier source', (v) => packInputDir = v)}>Parcourir</button></div>
                <label>Sortie</label>
                <div class="row"><input bind:value={packOutput} /><button on:click={() => saveFile('Archive de sortie', 'archive_custom', '*.*', 'Tous les fichiers', (v) => packOutput = v)}>Parcourir</button></div>
                <label>Clés</label>
                <div class="row compact"><input bind:value={packKey} placeholder="LST 01" /><input bind:value={packSnxKey} placeholder="SNX 02" /></div>
              </div>
              <div class="actions left">
                <button on:click={startPackArchive} disabled={running || !packInputDir || !packOutput}>Packer</button>
              </div>
            </div>
          </div>
        </section>

      {:else if selectedOp === 'hook'}
        <section class="form-view">
          <h1>Hook accents ONE / MOON</h1>
          <div class="block full">
            <div class="form-grid">
              <label>Jeu</label>
              <select bind:value={hookGame}>
                <option value="one">ONE</option>
                <option value="moon">MOON (archive anglaise)</option>
              </select>

              <label>Kit</label>
              <input value={hookDir || 'bin/one_hook'} readonly />

              <label>Police</label>
              <input bind:value={hookFontName} />

              <label>Debug</label>
              <select bind:value={hookDebugLog}>
                <option value="0">0</option>
                <option value="1">1</option>
                <option value="2">2</option>
              </select>

              <label>Dossier jeu</label>
              <div class="row"><input bind:value={hookGameDir} readonly /><button on:click={() => pickDir(`Dossier du jeu ${hookGame === 'moon' ? 'MOON' : 'ONE'}`, (v) => hookGameDir = v)}>Parcourir</button></div>
            </div>
            <div class="actions">
              <button on:click={reloadHookConfig} disabled={running}>Recharger</button>
              <button on:click={saveHookConfig} disabled={running}>Enregistrer</button>
              <button class="primary" on:click={installHook} disabled={running || !hookGameDir}>Installer</button>
            </div>
          </div>
        </section>

      {:else if selectedOp === 'about'}
        <section class="about">
          <img src={logo} alt="" />
          <h1>LCSE Tool GUI</h1>
          <p>Interface Wails/Svelte pour les workflows Nexton ONE et MOON.</p>
          <span>v1.2</span>
        </section>
      {/if}
    </main>
  </div>

  <footer class="console-wrapper">
    <div class="console-header">
      <span>Console</span>
      <div class="console-actions">
        {#if running}<button class="stop" on:click={stopProcess}>Stop</button>{/if}
        <button on:click={clearConsole}>Clear</button>
      </div>
    </div>
    <div class="console" bind:this={consoleEl}>
      {#each consoleLines as line}
        <div class={line.cls}>{line.text}</div>
      {/each}
    </div>
  </footer>
</div>
