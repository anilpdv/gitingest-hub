// Chrome Built-in Native AI & Comprehensive Clipboard Utilities

let aiSession = null;
let aiAvailable = false;

// Check Chrome Native AI availability
async function checkChromeAIAvailability() {
  const statusEl = document.getElementById("chrome-ai-status");
  if (!statusEl) return;

  try {
    if (window.ai && window.ai.languageModel) {
      const capabilities = await window.ai.languageModel.capabilities();
      if (capabilities && capabilities.available !== "no") {
        aiAvailable = true;
        statusEl.className = "nb-badge nb-card-lime";
        statusEl.innerHTML = `<svg class="w-4 h-4 inline mr-1" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13 10V3L4 14h7v7l9-11h-7z"/></svg><span>Chrome Built-in AI (Gemini Nano) Active</span>`;
        return;
      }
    } else if (window.ai && window.ai.summarizer) {
      const capabilities = await window.ai.summarizer.capabilities();
      if (capabilities && capabilities.available !== "no") {
        aiAvailable = true;
        statusEl.className = "nb-badge nb-card-lime";
        statusEl.innerHTML = `<svg class="w-4 h-4 inline mr-1" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13 10V3L4 14h7v7l9-11h-7z"/></svg><span>Chrome Built-in AI (Summarizer) Active</span>`;
        return;
      }
    }
  } catch (err) {
    console.warn("Chrome AI check error:", err);
  }

  aiAvailable = false;
  statusEl.className = "nb-badge nb-card-yellow";
  statusEl.innerHTML = `<svg class="w-4 h-4 inline mr-1" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"/></svg><span>Chrome AI Ready (Gemini Nano via chrome://flags)</span>`;
}

// Initialize on page load
document.addEventListener("DOMContentLoaded", () => {
  checkChromeAIAvailability();
  initRecentRepos();
});

// ==========================================
// LocalStorage Recent Repositories Manager
// ==========================================
const RECENT_REPOS_KEY = "gitingest_recent_repos";

function getRecentRepos() {
  try {
    const raw = localStorage.getItem(RECENT_REPOS_KEY);
    return raw ? JSON.parse(raw) : [];
  } catch (e) {
    console.warn("Error reading recent repos from localStorage:", e);
    return [];
  }
}

function saveRecentRepo(url, branch = "") {
  if (!url || typeof url !== "string") return;
  url = url.trim();
  if (!url) return;

  try {
    let repos = getRecentRepos().filter(r => r.url.toLowerCase() !== url.toLowerCase());
    repos.unshift({ url, branch: branch.trim(), timestamp: Date.now() });
    repos = repos.slice(0, 8); // Keep up to 8 recent repos
    localStorage.setItem(RECENT_REPOS_KEY, JSON.stringify(repos));
    renderRecentRepos();
  } catch (e) {
    console.warn("Error saving recent repo to localStorage:", e);
  }
}

function removeRecentRepo(event, url) {
  event.stopPropagation();
  try {
    let repos = getRecentRepos().filter(r => r.url.toLowerCase() !== url.toLowerCase());
    localStorage.setItem(RECENT_REPOS_KEY, JSON.stringify(repos));
    renderRecentRepos();
  } catch (e) {
    console.warn("Error removing recent repo:", e);
  }
}

function renderRecentRepos() {
  const container = document.getElementById("recent-repos-list");
  if (!container) return;

  const repos = getRecentRepos();
  if (!repos.length) {
    container.innerHTML = `<span class="mono text-xs text-gray-400 italic">No recent repos yet</span>`;
    return;
  }

  container.innerHTML = repos.map(r => `
    <div class="chip flex items-center gap-1.5 py-1 px-2.5 group hover:bg-yellow-300 transition-colors" 
         onclick="setRepo('${r.url}', '${r.branch || ""}')"
         title="Click to load ${r.url}">
      <span class="truncate max-w-[200px] font-bold">${r.url.replace(/^https?:\/\/(www\.)?github\.com\//, "")}</span>
      ${r.branch ? `<span class="text-[10px] bg-black text-white px-1 rounded">${r.branch}</span>` : ""}
      <button type="button" 
              onclick="removeRecentRepo(event, '${r.url}')" 
              title="Remove from recents"
              class="opacity-40 hover:opacity-100 hover:text-red-600 ml-0.5 text-xs font-black">
        &times;
      </button>
    </div>
  `).join("");
}

function initRecentRepos() {
  renderRecentRepos();

  const form = document.getElementById("ingest-form");
  if (form) {
    form.addEventListener("submit", () => {
      const urlInput = document.getElementById("repo-url-input");
      const branchInput = document.getElementById("repo-branch-input");
      if (urlInput && urlInput.value) {
        saveRecentRepo(urlInput.value, branchInput ? branchInput.value : "");
      }
    });
  }
}

// Dynamic split mode toggle for rectangle box
function setSplitMode(prefix, mode) {
  const panels = document.querySelectorAll(`.${prefix}-split-panel`);
  panels.forEach(p => p.style.display = "none");

  const target = document.getElementById(`${prefix}-panel-${mode}`);
  if (target) {
    target.style.display = "flex";
  }
}

// Set Repo Input from preset chips
function setRepo(url, branch = "") {
  const input = document.getElementById("repo-url-input");
  const branchInput = document.getElementById("repo-branch-input");
  if (input) {
    input.value = url;
    if (branchInput) branchInput.value = branch;
    input.focus();
  }
}

// Filter files inside an individual chunk card
function filterChunkFiles(chunkIndex, query) {
  const container = document.getElementById(`chunk-file-list-${chunkIndex}`);
  if (!container) return;

  const tags = container.querySelectorAll(".chunk-file-tag");
  const q = query.toLowerCase().trim();

  tags.forEach(tag => {
    const path = tag.getAttribute("data-chunk-path") || "";
    if (!q || path.toLowerCase().includes(q)) {
      tag.style.display = "inline-flex";
    } else {
      tag.style.display = "none";
    }
  });
}

// Generic Clipboard Copy Helper with Tactile Feedback
function copyToClipboard(text, btnElement, successLabel = "COPIED!") {
  if (!text) {
    console.warn("copyToClipboard called with empty text");
    return;
  }

  const fallbackCopy = () => {
    try {
      const textArea = document.createElement("textarea");
      textArea.value = text;
      textArea.style.position = "fixed";
      textArea.style.left = "-999999px";
      textArea.style.top = "-999999px";
      document.body.appendChild(textArea);
      textArea.focus();
      textArea.select();
      const successful = document.execCommand("copy");
      document.body.removeChild(textArea);
      if (successful) {
        showSuccessState(btnElement, successLabel);
      }
    } catch (err) {
      console.error("Fallback clipboard copy failed:", err);
    }
  };

  if (navigator.clipboard && window.isSecureContext) {
    navigator.clipboard.writeText(text).then(() => {
      showSuccessState(btnElement, successLabel);
    }).catch(err => {
      console.warn("navigator.clipboard failed, attempting execCommand fallback:", err);
      fallbackCopy();
    });
  } else {
    fallbackCopy();
  }
}

function showSuccessState(btnElement, successLabel) {
  if (!btnElement) return;
  const originalHTML = btnElement.innerHTML;
  btnElement.innerHTML = `<svg class="w-3.5 h-3.5 inline mr-1" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7"/></svg><span>${successLabel}</span>`;
  btnElement.classList.add("nb-btn-lime");

  setTimeout(() => {
    btnElement.innerHTML = originalHTML;
    btnElement.classList.remove("nb-btn-lime");
  }, 1800);
}

// 1-Click Copy Chunk Text
function copyChunkText(buttonId, chunkIndex) {
  const codeEl = document.getElementById(`chunk-code-${chunkIndex}`);
  const btn = document.getElementById(buttonId);
  if (!codeEl || !btn) return;
  const text = codeEl.textContent || codeEl.innerText;
  copyToClipboard(text, btn, "CHUNK COPIED!");
}

// Global Cache of LLM-ready context summaries per chunk
window.__chunkSummaries = window.__chunkSummaries || {};

// 1-Click Copy Summary Text (Optimized specifically for LLMs / Agents)
function copySummaryText(buttonId, chunkIndex) {
  const btn = document.getElementById(buttonId);
  const llmContext = window.__chunkSummaries[chunkIndex];

  if (llmContext) {
    copyToClipboard(llmContext, btn, "LLM CONTEXT COPIED!");
    return;
  }

  // Fallback to text content if markdown summary wasn't found
  const summaryEl = document.getElementById(`ai-output-${chunkIndex}`) || document.getElementById(`chunk-ai-box-${chunkIndex}`);
  if (!summaryEl || !btn) return;
  const text = summaryEl.textContent || summaryEl.innerText;
  copyToClipboard(text, btn, "SUMMARY COPIED!");
}

// 1-Click Copy Visual Summary (Plain Text)
function copyVisualSummaryText(buttonId, chunkIndex) {
  const summaryEl = document.getElementById(`ai-output-${chunkIndex}`) || document.getElementById(`chunk-ai-box-${chunkIndex}`);
  const btn = document.getElementById(buttonId);
  if (!summaryEl || !btn) return;
  const text = summaryEl.innerText || summaryEl.textContent;
  copyToClipboard(text, btn, "COPIED!");
}

// 1-Click Copy Directory Tree
function copyTreeText(buttonId) {
  const treeEl = document.getElementById("directory-tree-text");
  const btn = document.getElementById(buttonId);
  if (!treeEl || !btn) {
    console.error("Tree element or button not found:", { treeEl, btn });
    return;
  }
  // Use textContent because innerText returns empty string if parent <details> is closed!
  const treeText = treeEl.textContent || treeEl.innerText;
  copyToClipboard(treeText, btn, "TREE COPIED!");
}

// 1-Click Copy Individual File Path
function copyFilePath(path, buttonId) {
  const btn = document.getElementById(buttonId);
  copyToClipboard(path, btn, "PATH COPIED!");
}

// 1-Click Copy Individual File Content
function copyFileContent(fileIndex, buttonId) {
  const templateEl = document.getElementById(`file-content-${fileIndex}`);
  const btn = document.getElementById(buttonId);
  if (!templateEl || !btn) return;
  const content = templateEl.innerHTML || templateEl.textContent;
  copyToClipboard(content, btn, "CODE COPIED!");
}

// Filter File List Live in Search Input
function filterFileList(query) {
  const rows = document.querySelectorAll(".file-row");
  const q = query.toLowerCase().trim();
  rows.forEach(row => {
    const path = row.getAttribute("data-path") || "";
    if (!q || path.toLowerCase().includes(q)) {
      row.style.display = "flex";
    } else {
      row.style.display = "none";
    }
  });
}

// Chrome Native AI Summarizer
async function summarizeChunk(chunkIndex) {
  const codeEl = document.getElementById(`chunk-code-${chunkIndex}`);
  const targetBox = document.getElementById(`chunk-ai-box-${chunkIndex}`);
  const btn = document.getElementById(`btn-ai-summarize-${chunkIndex}`);
  if (!codeEl || !targetBox) return;

  targetBox.style.display = "block";
  targetBox.innerHTML = `
    <div class="flex items-center gap-2 font-bold mono text-xs mb-2 text-indigo-900">
      <svg class="w-4 h-4 animate-spin text-black" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15"/></svg>
      <span>Analyzing chunk architecture & types in-browser...</span>
    </div>
  `;

  const fullText = codeEl.textContent || codeEl.innerText;
  const sampleCode = fullText.slice(0, 16000);

  try {
    if (window.ai && window.ai.languageModel) {
      if (!aiSession) {
        aiSession = await window.ai.languageModel.create({
          systemPrompt: "You are a principal software architect. Provide a high-density, structured summary of this code chunk. List: 1. Core Purpose & Architectural Role (1-2 sentences), 2. Key Types & Public APIs, 3. Concurrency / State Handling, 4. Dependencies. Keep it crisp, technical, and scannable with bullet points."
        });
      }

      const prompt = `Analyze and summarize this repository code chunk:\n\n${sampleCode}`;
      const stream = aiSession.promptStreaming(prompt);
      
      targetBox.innerHTML = `
        <div class="flex items-center justify-between pb-2 mb-3 border-b-2 border-black">
          <div class="flex items-center gap-2">
            <span class="nb-badge nb-card-lime text-xs font-black">CHROME GEMINI NANO</span>
            <span class="font-extrabold text-sm text-black">Architectural Analysis</span>
          </div>
          <button id="btn-copy-summary-${chunkIndex}" onclick="copySummaryText('btn-copy-summary-${chunkIndex}', ${chunkIndex})" class="nb-btn nb-btn-white text-xs py-1 px-2.5 flex items-center gap-1">
            <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 16H6a2 2 0 01-2-2V6a2 2 0 012-2h8a2 2 0 012 2v2m-6 12h8a2 2 0 002-2v-8a2 2 0 00-2-2h-8a2 2 0 00-2 2v8a2 2 0 002 2z"/></svg>
            <span>Copy Context</span>
          </button>
        </div>
        <div class="mono text-xs leading-relaxed whitespace-pre-wrap text-gray-900 bg-white p-3 border border-black rounded shadow-[2px_2px_0px_#121212]" id="ai-output-${chunkIndex}"></div>
      `;
      const outputEl = document.getElementById(`ai-output-${chunkIndex}`);

      let fullGeneratedText = "";
      for await (const chunk of stream) {
        fullGeneratedText += chunk;
        outputEl.innerText = fullGeneratedText;
      }
      window.__chunkSummaries[chunkIndex] = fullGeneratedText;

      if (btn) btn.innerHTML = `<svg class="w-3.5 h-3.5 mr-1 inline" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15"/></svg><span>Re-Summarize</span>`;
      return;
    }
  } catch (err) {
    console.warn("Chrome AI streaming error, falling back to client AST analysis:", err);
  }

  // Comprehensive Smart Client-Side Architectural Analyzer
  const summaryHTML = generateArchitecturalSummary(fullText, chunkIndex);
  targetBox.innerHTML = summaryHTML;
}

// Deep Codebase Analyzer & Architectural Summary Generator
function generateArchitecturalSummary(fullText, chunkIndex) {
  // 1. Extract Files across all formats (Standard delimiters, XML tags, AST headers)
  const fileRegex = /(?:================================================\s*\n\s*FILE:\s*([^\n\r=]+)|<file\s+path=["']([^"']+)["']|#\s*---\s*FILE:\s*([^\n\r]+))/g;
  const fileMatches = [];
  let fileMatch;
  while ((fileMatch = fileRegex.exec(fullText)) !== null) {
    const p = (fileMatch[1] || fileMatch[2] || fileMatch[3] || "").trim();
    if (p && !fileMatches.includes(p)) {
      fileMatches.push(p);
    }
  }

  // Fallback if delimiters were stripped
  if (fileMatches.length === 0) {
    const simpleFileRegex = /FILE:\s*([^\n\r]+)/g;
    while ((fileMatch = simpleFileRegex.exec(fullText)) !== null) {
      const p = fileMatch[1].trim();
      if (p && !fileMatches.includes(p)) {
        fileMatches.push(p);
      }
    }
  }

  // 2. Identify Subsystems & Group Files into Architectural Modules
  const moduleMap = {};
  fileMatches.forEach(filePath => {
    const parts = filePath.split("/");
    let group = "Core";
    if (parts.length > 1) {
      if (parts.length >= 3 && (parts[0].includes("Domain") || parts[1].includes("Domain") || parts[0].includes("Climb") || parts[1].includes("Climb"))) {
        group = parts.slice(parts.length - 3, parts.length - 1).join("/");
      } else {
        group = parts.slice(0, parts.length - 1).join("/");
      }
    }
    if (!moduleMap[group]) moduleMap[group] = [];
    moduleMap[group].push(parts[parts.length - 1]);
  });

  const sortedModules = Object.entries(moduleMap)
    .sort((a, b) => b[1].length - a[1].length)
    .slice(0, 6);

  // 3. Extract Types, Protocols, Structs, Classes, Interfaces, Enums with Categorization
  const typeRegex = /(?:public\s+|final\s+|open\s+|internal\s+|export\s+|abstract\s+)*(struct|class|actor|enum|protocol|interface|type)\s+([A-Z][A-Za-z0-9_]+)/g;
  const typesByCategory = {
    protocols: new Set(),
    structs: new Set(),
    classes: new Set(),
    enums: new Set(),
    all: new Set()
  };

  let match;
  while ((match = typeRegex.exec(fullText)) !== null) {
    const kind = match[1];
    const name = match[2];
    if (name.length > 2 && !["String", "Int", "Double", "Float", "Bool", "Error", "View", "Any", "Self", "Result", "Tests", "TestCase", "XCTestCase", "Mock"].includes(name)) {
      typesByCategory.all.add(name);
      if (kind === "protocol" || kind === "interface") {
        typesByCategory.protocols.add(name);
      } else if (kind === "enum") {
        typesByCategory.enums.add(name);
      } else if (kind === "struct" || kind === "type") {
        typesByCategory.structs.add(name);
      } else {
        typesByCategory.classes.add(name);
      }
    }
  }

  const allTypes = Array.from(typesByCategory.all);
  const protocols = Array.from(typesByCategory.protocols);
  const dataModels = Array.from(typesByCategory.structs).concat(Array.from(typesByCategory.classes));
  const enums = Array.from(typesByCategory.enums);

  // 4. Extract Imports / Frameworks
  const importRegex = /(?:import\s+(?:typealias\s+|struct\s+|class\s+|enum\s+)?([A-Za-z0-9_]+)|from\s+['"]([A-Za-z0-9_@/-]+)['"])/g;
  const importsFound = new Set();
  while ((match = importRegex.exec(fullText)) !== null) {
    const imp = (match[1] || match[2] || "").trim();
    if (imp && !["Foundation", "os", "sys", "re", "fmt", "strings", "time"].includes(imp)) {
      importsFound.add(imp);
    }
  }
  const displayImports = Array.from(importsFound).slice(0, 8);

  // 5. Intelligent Multi-Domain Scoring
  const textLower = fullText.toLowerCase();
  
  const scores = {
    climb: (textLower.match(/\b(climb|elevation|gradient|altitude|slope|profilepoint|climbcategory|climbprogress)\b/g) || []).length,
    navigation: (textLower.match(/\b(navigation|route|waypoint|cue|bearing|heading|gps|turn|maneuver)\b/g) || []).length,
    metrics: (textLower.match(/\b(metric|cadence|power|heartrate|freshness|telemetry|sensor|speed)\b/g) || []).length,
    recovery: (textLower.match(/\b(failure|recover|diagnostic|degradation|severity|fault|rejection)\b/g) || []).length,
    coordinator: (textLower.match(/\b(coordinator|lifecycle|pipeline|orchestrat|dispatcher|engine)\b/g) || []).length,
    ui: (textLower.match(/\b(viewmodel|viewstate|swiftui|uikit|component|screen|dialog|presenter)\b/g) || []).length,
    infra: (textLower.match(/\b(bluetooth|healthkit|repository|storage|client|network|socket|adapter)\b/g) || []).length,
  };

  const activeDomains = Object.entries(scores)
    .filter(([_, score]) => score >= 3)
    .sort((a, b) => b[1] - a[1]);

  let domainDescription = "";
  let architectureSummaryBullets = [];

  const domainSummaries = {
    climb: {
      name: "⛰️ Climb Detection, Elevation & Gradient Profiles",
      plainName: "Climb & Elevation Engine",
      insight: "Implements real-time climb categorization, elevation slicing, gradient calculation, and ascent rate tracking."
    },
    navigation: {
      name: "🧭 Spatial Navigation, Turn Cues & Route State",
      plainName: "Navigation & Spatial State Machine",
      insight: "Houses turn-by-turn waypoint tracking, bearing calculation, heading alignment, and navigation finite-state machines."
    },
    metrics: {
      name: "📊 Telemetry Aggregation, Ride Metrics & Freshness",
      plainName: "Telemetry & Sensor Pipeline",
      insight: "Processes live sensor streams (cadence, speed, power, heart rate) and monitors data freshness guarantees."
    },
    recovery: {
      name: "🛡️ Fault-Tolerance, Diagnostic Errors & Self-Healing",
      plainName: "Fault-Tolerance & Error Architecture",
      insight: "Defines granular failure codes, severity/impact ratings, and automated recovery actions for operational resilience."
    },
    coordinator: {
      name: "⚙️ Application Lifecycle & Multi-Engine Coordinators",
      plainName: "App Orchestration & Event Coordinators",
      insight: "Coordinates reactive event pipelines between sensors, business engines, and state dispatchers."
    },
    ui: {
      name: "🖥️ Reactive Presentation & UI State Binding",
      plainName: "Presentation & UI State Layer",
      insight: "Contains ViewModels, banner notifications, and view-state projections decoupled from underlying engines."
    },
    infra: {
      name: "🔌 Hardware SDK Bridges & Peripheral Adapters",
      plainName: "Hardware Adapters & Peripherals",
      insight: "Manages CoreBluetooth peripherals, HealthKit integration, and hardware sensor telemetry parsing."
    }
  };

  if (activeDomains.length > 1) {
    domainDescription = `Composite Domain Layer: Combines ${activeDomains.slice(0, 3).map(([d]) => domainSummaries[d]?.name || d).join(", ")}.`;
    architectureSummaryBullets = activeDomains.slice(0, 4).map(([d]) => `<strong>${domainSummaries[d]?.name || d}</strong>: ${domainSummaries[d]?.insight || "Encapsulates specialized domain logic."}`);
  } else if (activeDomains.length === 1) {
    const d = activeDomains[0][0];
    domainDescription = domainSummaries[d]?.name || "Core Domain Architecture";
    architectureSummaryBullets = [
      domainSummaries[d]?.insight || "Pure immutable domain entities and business rules.",
      "Strict separation between business invariants and infrastructure side effects.",
      "Thread-safe data structures suitable for concurrent pipelines."
    ];
  } else {
    domainDescription = "📐 Immutable Domain Entities & State Contracts";
    architectureSummaryBullets = [
      "Pure domain entities, value objects, and snapshot records.",
      "Zero side-effect model hierarchy isolated from I/O and UI dependencies.",
      "Thread-safe data contracts designed for concurrent state exchange."
    ];
  }

  // Primary sample types for LLM prompt
  const focusTypes = allTypes.slice(0, 5).join(", ") || (sortedModules[0] ? sortedModules[0][0] : "core models");
  const promptSuggestion = `Analyze the architectural boundaries and concurrency safety of this code chunk (focusing on ${focusTypes}). Verify whether state mutations are pure and identify edge-case recovery gaps.`;

  // Build Comprehensive LLM / Agent Context Payload (Optimized for Claude 3.5 Sonnet, GPT-4o, Gemini 1.5 Pro)
  const llmContextMarkdown = [
    `# ARCHITECTURAL SUMMARY & CONTEXT: Part ${chunkIndex}`,
    `**Domain Role**: ${domainDescription.replace(/<[^>]+>/g, "")}`,
    `**Files Contained** (${fileMatches.length} files, ${Math.round(fullText.length / 1024)} KB):`,
    fileMatches.slice(0, 20).map(f => `- ${f}`).join("\n") + (fileMatches.length > 20 ? `\n- ... and ${fileMatches.length - 20} more files` : ""),
    ``,
    `## Core Subsystems & Modules:`,
    sortedModules.map(([dir, files]) => `- **${dir}** (${files.length} files): ${files.slice(0, 5).join(", ")}${files.length > 5 ? ", ..." : ""}`).join("\n"),
    ``,
    `## Key Types, Protocols & Structs:`,
    protocols.length > 0 ? `- **Protocols / Interfaces**: ${protocols.join(", ")}` : "",
    dataModels.length > 0 ? `- **Core Entities & Models**: ${dataModels.slice(0, 20).join(", ")}` : "",
    enums.length > 0 ? `- **Enums / State Codes**: ${enums.slice(0, 10).join(", ")}` : "",
    displayImports.length > 0 ? `- **External Dependencies**: ${displayImports.join(", ")}` : "",
    ``,
    `## Architectural Invariants & Behavior:`,
    activeDomains.length > 0
      ? activeDomains.map(([d]) => `- **${domainSummaries[d]?.plainName || d}**: ${domainSummaries[d]?.insight || ""}`).join("\n")
      : `- Pure immutable domain models without side effects.\n- Thread-safe value objects suitable for concurrent state exchange.`
  ].filter(line => line !== "").join("\n");

  // Save to global context cache for instant high-yield copying
  window.__chunkSummaries[chunkIndex] = llmContextMarkdown;

  return `
    <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-2 pb-3 mb-3 border-b-2 border-black">
      <div class="flex items-center gap-2 flex-wrap">
        <span class="nb-badge nb-card-yellow text-xs font-black tracking-wider uppercase">DEEP ARCHITECTURE SUMMARY</span>
        <span class="font-extrabold text-sm md:text-base text-black">${domainDescription}</span>
      </div>
      <div class="flex items-center gap-2 self-start sm:self-auto flex-shrink-0">
        <button id="btn-copy-llm-${chunkIndex}" onclick="copySummaryText('btn-copy-llm-${chunkIndex}', ${chunkIndex})" class="nb-btn nb-btn-lime text-xs py-1 px-3 flex items-center gap-1.5 shadow-[2px_2px_0px_#121212]" title="Copy structured Markdown context optimized for LLMs">
          <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13 10V3L4 14h7v7l9-11h-7z"/></svg>
          <span>Copy LLM Context</span>
        </button>
      </div>
    </div>

    <div class="mono text-xs leading-relaxed space-y-4" id="ai-output-${chunkIndex}">
      
      <!-- Architectural Highlights Card -->
      <div class="bg-white p-3.5 border-2 border-black rounded shadow-[3px_3px_0px_#121212] space-y-2">
        <div class="flex items-center justify-between text-gray-950 font-black text-xs">
          <span>Key Subsystems & Domain Invariants:</span>
          <span class="text-gray-500 font-bold">${fileMatches.length} files • ${Math.round(fullText.length / 1024)} KB</span>
        </div>
        <ul class="list-disc pl-4 space-y-1.5 text-gray-800 text-[11px] leading-relaxed">
          ${architectureSummaryBullets.map(b => `<li>${b}</li>`).join("")}
        </ul>
      </div>

      <!-- Subsystem Module Breakdown -->
      ${sortedModules.length > 0 ? `
      <div>
        <span class="font-black text-gray-900 uppercase tracking-wider text-[11px] block mb-1.5 flex items-center gap-1">
          <svg class="w-3.5 h-3.5 text-black" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M3 7v10a2 2 0 002 2h14a2 2 0 002-2V9a2 2 0 00-2-2h-6l-2-2H5a2 2 0 00-2 2z"/></svg>
          Module Clustering & Subsystems:
        </span>
        <div class="grid grid-cols-1 sm:grid-cols-2 md:grid-cols-3 gap-2">
          ${sortedModules.map(([dir, files]) => `
            <div class="p-2 bg-white border-2 border-black rounded shadow-[2px_2px_0px_#121212]">
              <div class="font-extrabold text-black text-xs truncate flex items-center justify-between">
                <span>📁 ${dir}</span>
                <span class="text-[10px] bg-yellow-200 px-1.5 rounded font-bold">${files.length}</span>
              </div>
              <p class="text-[10px] text-gray-600 truncate mt-0.5">${files.slice(0, 3).join(", ")}${files.length > 3 ? "..." : ""}</p>
            </div>
          `).join("")}
        </div>
      </div>` : ""}

      <!-- Protocols & Public Contracts -->
      ${protocols.length > 0 ? `
      <div>
        <span class="font-black text-gray-900 uppercase tracking-wider text-[11px] block mb-1.5 flex items-center gap-1">
          <svg class="w-3.5 h-3.5 text-purple-700" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12l2 2 4-4m5.618-4.016A11.955 11.955 0 0112 2.944a11.955 11.955 0 01-8.618 3.04A12.02 12.02 0 003 9c0 5.591 3.824 10.29 9 11.622 5.176-1.332 9-6.03 9-11.622 0-1.042-.133-2.052-.382-3.016z"/></svg>
          Abstract Protocols & Interfaces (${protocols.length}):
        </span>
        <div class="flex flex-wrap gap-1.5">
          ${protocols.map(p => `<span class="nb-badge nb-card-purple text-xs py-0.5 px-2 font-mono font-bold">${p}</span>`).join("")}
        </div>
      </div>` : ""}

      <!-- Core Structs & Domain Models -->
      ${dataModels.length > 0 ? `
      <div>
        <div class="flex items-center justify-between mb-1.5">
          <span class="font-black text-gray-900 uppercase tracking-wider text-[11px] flex items-center gap-1">
            <svg class="w-3.5 h-3.5 text-blue-700" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 11H5m14 0a2 2 0 012 2v6a2 2 0 01-2 2H5a2 2 0 01-2-2v-6a2 2 0 012-2m14 0V9a2 2 0 00-2-2M5 11V9a2 2 0 012-2m0 0V5a2 2 0 012-2h6a2 2 0 012 2v2M7 7h10"/></svg>
            Key Entities & Data Models (${dataModels.length}):
          </span>
        </div>
        <div class="flex flex-wrap gap-1.5">
          ${dataModels.slice(0, 16).map(t => `<span class="nb-badge nb-card-cyan text-xs py-0.5 px-2 font-mono font-bold">${t}</span>`).join("")}
          ${dataModels.length > 16 ? `<span class="nb-badge bg-gray-100 text-xs py-0.5 px-1.5 font-bold text-gray-600">+${dataModels.length - 16} more</span>` : ""}
        </div>
      </div>` : ""}

      <!-- Frameworks & Dependencies -->
      ${displayImports.length > 0 ? `
      <div>
        <span class="font-black text-gray-900 uppercase tracking-wider text-[11px] block mb-1.5 flex items-center gap-1">
          <svg class="w-3.5 h-3.5 text-green-700" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M20 7l-8-4-8 4m16 0l-8 4m8-4v10l-8 4m0-10L4 7m8 4v10M4 7v10l8 4"/></svg>
          Frameworks & Imports:
        </span>
        <div class="flex flex-wrap gap-1.5">
          ${displayImports.map(imp => `<span class="nb-badge nb-card-lime text-xs py-0.5 px-2 font-mono font-bold">${imp}</span>`).join("")}
        </div>
      </div>` : ""}
    </div>
  `;
}




