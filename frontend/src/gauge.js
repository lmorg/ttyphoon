export function drawGauge(offCtx, getCellSize, cmd) {
    if (!offCtx || !cmd?.fg || !Number.isFinite(cmd.max) || cmd.max <= 0) {
        return;
    }

    const { cellWidth, cellHeight } = getCellSize();

    const x = (Number.isFinite(cmd.x) ? cmd.x : 0) * cellWidth;
    const y = (Number.isFinite(cmd.y) ? cmd.y : 0) * cellHeight;
    const ratio = Math.max(0, Math.min(1, (Number.isFinite(cmd.value) ? cmd.value : 0) / cmd.max));

    const base = `rgb(${cmd.fg.Red}, ${cmd.fg.Green}, ${cmd.fg.Blue})`;

    if (cmd.op === 'gauge_h') {
        const widthCells = Number.isFinite(cmd.width) && cmd.width > 0 ? cmd.width : 1;
        const fullW = widthCells * cellWidth;

        offCtx.globalAlpha = 0.13;
        offCtx.fillStyle = base;
        offCtx.fillRect(x, y, fullW, cellHeight);

        offCtx.globalAlpha = 0.75;
        offCtx.fillRect(x, y, Math.floor(fullW * ratio), cellHeight);
        offCtx.globalAlpha = 1;
        return;
    }

    if (cmd.op === 'gauge_v') {
        const heightCells = Number.isFinite(cmd.height) && cmd.height > 0 ? cmd.height : 1;
        const fullH = heightCells * cellHeight;

        offCtx.globalAlpha = 0.13;
        offCtx.fillStyle = base;
        offCtx.fillRect(x, y, cellWidth, fullH);

        const fillH = Math.floor(fullH * ratio);
        offCtx.globalAlpha = 0.75;
        offCtx.fillRect(x, y, cellWidth, fillH);
        offCtx.globalAlpha = 1;
    }
}
