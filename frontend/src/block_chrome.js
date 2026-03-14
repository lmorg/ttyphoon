export function drawBlockChrome(offCtx, getCellSize, cmd) {
    if (!offCtx || !cmd?.fg) {
        return;
    }

    const { cellWidth, cellHeight } = getCellSize();

    const xCell = Number.isFinite(cmd.x) ? cmd.x : 0;
    const yCell = Number.isFinite(cmd.y) ? cmd.y : 0;
    const heightCells = Number.isFinite(cmd.height) && cmd.height > 0 ? cmd.height : 1;

    const x = 0; //xCell * cellWidth;
    const y = yCell * cellHeight;
    const h = heightCells * cellHeight;
    const barWidth = cellWidth / 2; //Math.max(2, Math.floor(cellWidth * (cmd.folded ? 0.5 : 0.25)));

    offCtx.fillStyle = `rgb(${cmd.fg.Red}, ${cmd.fg.Green}, ${cmd.fg.Blue})`;
    offCtx.globalAlpha = 0.75;
    offCtx.fillRect(x, y, barWidth, h);

    if (!cmd.folded && Number.isFinite(cmd.endX) && cmd.endX >= xCell) {
        const lineY = y + h;
        const lineEndX = ((cmd.endX + 1) * cellWidth) - 1;
        offCtx.fillRect(x, lineY, Math.max(1, lineEndX - x + 1), 1);
    }

    offCtx.globalAlpha = 1;
}
