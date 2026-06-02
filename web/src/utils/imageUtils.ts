export async function toWebP(file: File, quality = 0.85): Promise<File> {
  if (file.type === 'image/webp') return file
  return new Promise((resolve, reject) => {
    const img = new Image()
    const url = URL.createObjectURL(file)
    img.onload = () => {
      URL.revokeObjectURL(url)
      const canvas = document.createElement('canvas')
      canvas.width = img.naturalWidth
      canvas.height = img.naturalHeight
      const ctx = canvas.getContext('2d')!
      ctx.drawImage(img, 0, 0)
      canvas.toBlob(blob => {
        if (!blob) { reject(new Error('WebP conversion failed')); return }
        const newName = file.name.replace(/\.[^.]+$/, '') + '.webp'
        resolve(new File([blob], newName, { type: 'image/webp' }))
      }, 'image/webp', quality)
    }
    img.onerror = () => { URL.revokeObjectURL(url); reject(new Error('Image load failed')) }
    img.src = url
  })
}
