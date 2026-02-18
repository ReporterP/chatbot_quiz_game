import { useState, useRef, useEffect } from 'react';
import './MediaViewer.css';

function AudioPlayer({ src }) {
  const audioRef = useRef(null);
  const [playing, setPlaying] = useState(false);
  const [progress, setProgress] = useState(0);
  const [duration, setDuration] = useState(0);

  useEffect(() => {
    const a = audioRef.current;
    if (!a) return;
    const onTimeUpdate = () => setProgress(a.duration ? (a.currentTime / a.duration) * 100 : 0);
    const onLoaded = () => setDuration(a.duration || 0);
    const onEnded = () => setPlaying(false);
    a.addEventListener('timeupdate', onTimeUpdate);
    a.addEventListener('loadedmetadata', onLoaded);
    a.addEventListener('ended', onEnded);
    return () => {
      a.removeEventListener('timeupdate', onTimeUpdate);
      a.removeEventListener('loadedmetadata', onLoaded);
      a.removeEventListener('ended', onEnded);
    };
  }, []);

  const toggle = () => {
    const a = audioRef.current;
    if (!a) return;
    if (playing) a.pause(); else a.play();
    setPlaying(!playing);
  };

  const seek = (e) => {
    const a = audioRef.current;
    if (!a || !a.duration) return;
    const rect = e.currentTarget.getBoundingClientRect();
    const x = e.clientX - rect.left;
    a.currentTime = (x / rect.width) * a.duration;
  };

  const fmt = (s) => {
    if (!s || !isFinite(s)) return '0:00';
    const m = Math.floor(s / 60);
    const sec = Math.floor(s % 60);
    return `${m}:${sec.toString().padStart(2, '0')}`;
  };

  return (
    <div className="mv-audio">
      <audio ref={audioRef} src={src} preload="metadata" />
      <button className="mv-audio-btn" onClick={toggle} type="button">
        {playing ? '⏸' : '▶'}
      </button>
      <div className="mv-audio-track" onClick={seek}>
        <div className="mv-audio-progress" style={{ width: `${progress}%` }} />
      </div>
      <span className="mv-audio-time">{fmt(audioRef.current?.currentTime)}/{fmt(duration)}</span>
    </div>
  );
}

function VideoPlayer({ src, compact }) {
  return (
    <div className={`mv-video-wrap${compact ? ' compact' : ''}`}>
      <video
        src={src}
        controls
        preload="metadata"
        playsInline
        className="mv-video"
      />
    </div>
  );
}

function ImageViewer({ src, alt, onLightbox }) {
  return (
    <div className="mv-image-wrap" onClick={() => onLightbox && onLightbox(src)}>
      <img src={src} alt={alt || ''} className="mv-image" loading="lazy" />
      {onLightbox && <div className="mv-image-zoom">🔍</div>}
    </div>
  );
}

export function Lightbox({ src, onClose }) {
  if (!src) return null;
  return (
    <div className="mv-lightbox" onClick={onClose}>
      <button className="mv-lightbox-close" onClick={onClose}>✕</button>
      <img src={src} alt="" onClick={(e) => e.stopPropagation()} />
    </div>
  );
}

export default function MediaViewer({ items, onLightbox, compact }) {
  if (!items || items.length === 0) return null;

  return (
    <div className={`mv-container${compact ? ' compact' : ''}`}>
      {items.map((item, i) => {
        const key = item.id || i;
        if (item.type === 'audio') return <AudioPlayer key={key} src={item.url} />;
        if (item.type === 'video') return <VideoPlayer key={key} src={item.url} compact={compact} />;
        return <ImageViewer key={key} src={item.url} alt="" onLightbox={onLightbox} />;
      })}
    </div>
  );
}
