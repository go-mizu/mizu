import { useState, useRef, useCallback } from "react"
import { Volume2, Loader2 } from "lucide-react"
import { Button } from "@/components/ui/button"
import { ttsUrl } from "@/api/client"

interface AudioButtonProps {
  lang: string
  text: string
}

export function AudioButton({ lang, text }: AudioButtonProps) {
  const [playing, setPlaying] = useState(false)
  const audioRef = useRef<HTMLAudioElement | null>(null)

  const handlePlay = useCallback(() => {
    if (playing) {
      audioRef.current?.pause()
      setPlaying(false)
      return
    }

    const url = ttsUrl(lang, text)
    const audio = new Audio(url)
    audioRef.current = audio

    audio.addEventListener("ended", () => setPlaying(false))
    audio.addEventListener("error", () => setPlaying(false))

    setPlaying(true)
    audio.play().catch(() => setPlaying(false))
  }, [lang, text, playing])

  const disabled = !text || text.length > 200

  return (
    <Button
      variant="ghost"
      size="icon"
      onClick={handlePlay}
      disabled={disabled}
      title={disabled ? "Text must be 1-200 characters" : "Listen"}
      className="h-8 w-8"
    >
      {playing ? (
        <Loader2 className="h-4 w-4 animate-spin" />
      ) : (
        <Volume2 className="h-4 w-4" />
      )}
    </Button>
  )
}
