import { useState, useRef } from 'react'
import { ChevronLeft, ChevronRight } from 'lucide-react'
import type { ImageResult } from '../../types/ai'

interface ImageCarouselProps {
  images: ImageResult[]
  onImageClick?: (image: ImageResult) => void
}

export function ImageCarousel({ images, onImageClick }: ImageCarouselProps) {
  const [showLeftNav, setShowLeftNav] = useState(false)
  const [showRightNav, setShowRightNav] = useState(true)
  const trackRef = useRef<HTMLDivElement>(null)

  if (!images.length) return null

  const handleScroll = () => {
    if (!trackRef.current) return
    const { scrollLeft, scrollWidth, clientWidth } = trackRef.current
    setShowLeftNav(scrollLeft > 0)
    setShowRightNav(scrollLeft < scrollWidth - clientWidth - 10)
  }

  const scrollTo = (direction: 'left' | 'right') => {
    if (!trackRef.current) return
    const scrollAmount = trackRef.current.clientWidth * 0.8
    trackRef.current.scrollBy({
      left: direction === 'left' ? -scrollAmount : scrollAmount,
      behavior: 'smooth',
    })
  }

  return (
    <div className="image-carousel">
      {showLeftNav && (
        <button
          type="button"
          className="carousel-nav prev"
          onClick={() => scrollTo('left')}
          aria-label="Scroll left"
        >
          <ChevronLeft size={18} />
        </button>
      )}

      <div
        ref={trackRef}
        className="carousel-track"
        onScroll={handleScroll}
      >
        {images.map((img, i) => (
          <div
            key={i}
            className={`carousel-item ${i === 0 ? 'featured' : ''}`}
            onClick={() => onImageClick?.(img)}
            role="button"
            tabIndex={0}
            onKeyDown={(e) => {
              if (e.key === 'Enter' || e.key === ' ') {
                onImageClick?.(img)
              }
            }}
          >
            <img
              src={img.thumbnail_url || img.url}
              alt={img.title}
              loading="lazy"
            />
          </div>
        ))}
      </div>

      {showRightNav && images.length > 3 && (
        <button
          type="button"
          className="carousel-nav next"
          onClick={() => scrollTo('right')}
          aria-label="Scroll right"
        >
          <ChevronRight size={18} />
        </button>
      )}
    </div>
  )
}
