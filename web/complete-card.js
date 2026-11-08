// Complete card fronts bypass the card renderer: all text and borders are in the PNG.
export async function renderCompleteCard(card) {
  const image = new Image();
  image.src = card.completeImage;
  await image.decode();
  const canvas = document.createElement('canvas');
  canvas.width = 744; canvas.height = 1039;
  const ctx = canvas.getContext('2d');
  ctx.fillStyle = '#fff'; ctx.fillRect(0, 0, canvas.width, canvas.height);
  // Fit the entire front without cropping or changing its proportions.
  const scale = Math.min(canvas.width / image.naturalWidth, canvas.height / image.naturalHeight);
  const width = image.naturalWidth * scale, height = image.naturalHeight * scale;
  ctx.imageSmoothingEnabled = false;
  ctx.drawImage(image, (canvas.width - width) / 2, (canvas.height - height) / 2, width, height);
  const warnings = [];
  if (image.naturalWidth < 744 || image.naturalHeight < 1039) {
    warnings.push(`Source PNG is ${image.naturalWidth} × ${image.naturalHeight}; enlarging it does not add detail.`);
  }
  return {canvas, warnings};
}
