export function drawGauge(offCtx, getCellSize, cmd, foreground = cmd?.fg) {
    if (!offCtx || !cmd || !foreground || !Number.isFinite(cmd.max) || cmd.max <= 0) {
        return;
    }

    const horizontal = cmd.op === 'gauge_h';
    if (!horizontal && cmd.op !== 'gauge_v') {
        return;
    }
    const { cellWidth, cellHeight } = getCellSize();
    const x = (Number.isFinite(cmd.x) ? cmd.x : 0) * cellWidth;
    const y = (Number.isFinite(cmd.y) ? cmd.y : 0) * cellHeight;
    const ratio = Math.max(0, Math.min(1, (Number.isFinite(cmd.value) ? cmd.value : 0) / cmd.max));
    const cells = horizontal ? cmd.width : cmd.height;
    const extent = (Number.isFinite(cells) && cells > 0 ? cells : 1) * (horizontal ? cellWidth : cellHeight);
    const filled = Math.floor(extent * ratio);
    if (filled <= 0) {
        return;
    }

    const thickness = 8;
    offCtx.save();
    offCtx.globalAlpha = 0.5;
    offCtx.fillStyle = `rgb(${foreground.Red}, ${foreground.Green}, ${foreground.Blue})`;
    offCtx.beginPath();
    offCtx.roundRect(
        horizontal ? x : x + cellWidth - thickness,
        horizontal ? y + cellHeight - thickness : y,
        horizontal ? filled : thickness,
        horizontal ? thickness : filled,
        4,
    );
    offCtx.fill();
    offCtx.restore();
}
