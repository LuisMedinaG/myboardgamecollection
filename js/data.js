let gamesData = null;

async function loadGames() {
  if (gamesData) return gamesData;
  const resp = await fetch("data/games.json");
  const json = await resp.json();
  gamesData = json.games;

  // Merge with localStorage custom games
  const customGames = JSON.parse(localStorage.getItem("customGames") || "[]");
  gamesData = [...gamesData, ...customGames];

  return gamesData;
}
