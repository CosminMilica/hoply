/** A file the service rendered, with the name it gave it. */
export interface DownloadedFile {
  name: string
  blob: Blob
}

/** The file name a response's content-disposition gives, else the fallback. */
export function fileName(response: Response, fallback: string): string {
  return /filename="?([^";]+)"?/.exec(response.headers.get('content-disposition') ?? '')?.[1] ?? fallback
}

/** Saves a downloaded file the way a download link would. */
export function saveFile(file: DownloadedFile) {
  const url = URL.createObjectURL(file.blob)
  const link = document.createElement('a')
  link.href = url
  link.download = file.name
  link.click()
  URL.revokeObjectURL(url)
}
