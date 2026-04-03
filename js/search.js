const STEPS = ["genre", "subgenre", "players", "playtime"];

const LABELS = {
  genre: "Genre",
  subgenre: "Sub-genre",
  players: "Number of players",
  playtime: "Play time",
};

function getParams() {
  return Object.fromEntries(new URLSearchParams(window.location.search));
}

function filterGames(games, params) {
  return games.filter((g) => {
    if (params.genre && g.genre !== params.genre) return false;
    if (params.subgenre && g.subgenre !== params.subgenre) return false;
    if (params.players && (parseInt(params.players) < g.players.min || parseInt(params.players) > g.players.max)) return false;
    if (params.playtime && !matchPlaytime(g.playtime, params.playtime)) return false;
    return true;
  });
}

function matchPlaytime(time, bucket) {
  if (bucket === "< 30 min") return time < 30;
  if (bucket === "30-60 min") return time >= 30 && time <= 60;
  if (bucket === "> 60 min") return time > 60;
  return true;
}

function currentStep(params) {
  for (const step of STEPS) {
    if (!params[step]) return step;
  }
  return null;
}

function getOptions(games, step) {
  if (step === "genre") {
    const genres = [...new Set(games.map((g) => g.genre))].sort();
    return genres.map((v) => ({
      value: v,
      label: v.charAt(0).toUpperCase() + v.slice(1),
      count: games.filter((g) => g.genre === v).length,
    }));
  }

  if (step === "subgenre") {
    const subs = [...new Set(games.map((g) => g.subgenre))].sort();
    return subs.map((v) => ({
      value: v,
      label: v.split("-").map((w) => w.charAt(0).toUpperCase() + w.slice(1)).join(" "),
      count: games.filter((g) => g.subgenre === v).length,
    }));
  }

  if (step === "players") {
    const counts = new Set();
    games.forEach((g) => {
      for (let i = g.players.min; i <= g.players.max; i++) counts.add(i);
    });
    return [...counts].sort((a, b) => a - b).map((n) => ({
      value: String(n),
      label: n === 1 ? "1 player" : `${n} players`,
      count: games.filter((g) => n >= g.players.min && n <= g.players.max).length,
    }));
  }

  if (step === "playtime") {
    const buckets = ["< 30 min", "30-60 min", "> 60 min"];
    return buckets
      .map((b) => ({
        value: b,
        label: b,
        count: games.filter((g) => matchPlaytime(g.playtime, b)).length,
      }))
      .filter((o) => o.count > 0);
  }

  return [];
}

function buildBreadcrumb(params, container) {
  const crumbs = [{ label: "Home", href: "index.html" }];
  const accum = {};

  for (const step of STEPS) {
    if (!params[step]) break;
    accum[step] = params[step];
    crumbs.push({
      label: LABELS[step] + ": " + params[step],
      href: "search.html?" + new URLSearchParams(accum).toString(),
    });
  }

  container.innerHTML = "";
  crumbs.forEach((c, i) => {
    if (i > 0) {
      const sep = document.createElement("span");
      sep.textContent = " > ";
      container.appendChild(sep);
    }
    if (i < crumbs.length - 1) {
      const a = document.createElement("a");
      a.href = c.href;
      a.textContent = c.label;
      container.appendChild(a);
    } else {
      const span = document.createElement("span");
      span.textContent = c.label;
      container.appendChild(span);
    }
  });
}

function buildBackHref(params) {
  const keys = STEPS.filter((s) => params[s]);
  if (keys.length === 0) return "index.html";
  const prev = {};
  for (let i = 0; i < keys.length - 1; i++) {
    prev[keys[i]] = params[keys[i]];
  }
  if (Object.keys(prev).length === 0) return "index.html";
  return "search.html?" + new URLSearchParams(prev).toString();
}

async function initSearch() {
  const games = await loadGames();
  const params = getParams();
  const filtered = filterGames(games, params);

  // If one game left, go to its page
  if (filtered.length === 1) {
    window.location.replace("game.html?id=" + filtered[0].id);
    return;
  }

  // If no games, show message
  if (filtered.length === 0) {
    document.getElementById("search-content").innerHTML =
      '<p style="text-align:center;color:var(--text-secondary);padding:2rem 0;">No games match these filters.</p>';
    document.getElementById("back-btn").href = buildBackHref(params);
    buildBreadcrumb(params, document.getElementById("breadcrumb"));
    return;
  }

  const step = currentStep(params);

  // If all filters applied but multiple games, show list
  if (!step) {
    showGameList(filtered);
    document.getElementById("back-btn").href = buildBackHref(params);
    buildBreadcrumb(params, document.getElementById("breadcrumb"));
    return;
  }

  // Check if this step has only one option — skip it
  const options = getOptions(filtered, step);
  if (options.length === 1) {
    const newParams = { ...params, [step]: options[0].value };
    window.location.replace("search.html?" + new URLSearchParams(newParams).toString());
    return;
  }

  // Render step
  document.getElementById("back-btn").href = buildBackHref(params);
  buildBreadcrumb(params, document.getElementById("breadcrumb"));

  const title = document.getElementById("step-title");
  title.textContent = LABELS[step];

  const list = document.getElementById("option-list");
  list.innerHTML = "";

  options.forEach((opt) => {
    const newParams = { ...params, [step]: opt.value };
    const href = "search.html?" + new URLSearchParams(newParams).toString();
    const a = document.createElement("a");
    a.className = "option-btn";
    a.href = href;
    a.innerHTML = `<span>${opt.label}</span><span class="option-count">${opt.count}</span>`;
    list.appendChild(a);
  });
}

function showGameList(games) {
  const content = document.getElementById("search-content");
  const title = document.getElementById("step-title");
  title.textContent = `${games.length} games found`;

  const list = document.getElementById("option-list");
  list.innerHTML = "";

  games
    .sort((a, b) => a.name.localeCompare(b.name))
    .forEach((g) => {
      const a = document.createElement("a");
      a.className = "game-list-item";
      a.href = "game.html?id=" + g.id;
      a.innerHTML = `
        <div>
          <div class="game-list-name">${g.name}</div>
          <div class="game-list-meta">${g.players.min}-${g.players.max} players &middot; ${g.playtime} min</div>
        </div>`;
      list.appendChild(a);
    });
}

initSearch();
