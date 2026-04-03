let gamesData = null;

async function loadGames() {
  if (gamesData) return gamesData;
  const resp = await fetch("data/games.json");
  const json = await resp.json();
  gamesData = json.games;
  return gamesData;
}
