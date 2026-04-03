async function initGame() {
  const params = new URLSearchParams(window.location.search);
  const id = params.get("id");

  if (!id) {
    window.location.replace("index.html");
    return;
  }

  const games = await loadGames();
  const game = games.find((g) => g.id === id);

  if (!game) {
    document.getElementById("game-content").innerHTML =
      '<p style="text-align:center;color:var(--text-secondary);padding:2rem 0;">Game not found.</p>';
    return;
  }

  document.title = game.name + " — My Board Game Collection";
  document.getElementById("game-title").textContent = game.name;

  const meta = document.getElementById("game-meta");
  meta.innerHTML = `
    <span class="tag">${game.genre}</span>
    <span class="tag">${game.subgenre.split("-").map(w => w.charAt(0).toUpperCase() + w.slice(1)).join(" ")}</span>
    <span class="tag">${game.players.min}-${game.players.max} players</span>
    <span class="tag">${game.playtime} min</span>
  `;

  document.getElementById("quickref-text").textContent = game.quickref;

  if (game.rulesUrl) {
    const link = document.getElementById("rules-link");
    link.href = game.rulesUrl;
    link.style.display = "block";
  }
}

initGame();
