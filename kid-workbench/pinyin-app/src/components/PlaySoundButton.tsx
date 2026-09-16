import { useEffect, useRef, useState } from "react";

type PlaySoundButtonProps = {
  playing?: boolean;
  onPlay?: () => void;
  className?: string;
};

export function PlaySoundButton({
  playing = false,
  onPlay,
  className = "",
}: PlaySoundButtonProps) {
  const [burst, setBurst] = useState(false);
  const burstTimer = useRef<number>(0);

  useEffect(() => () => window.clearTimeout(burstTimer.current), []);

  function handlePlay() {
    onPlay?.();
    window.clearTimeout(burstTimer.current);
    setBurst(true);
    burstTimer.current = window.setTimeout(() => setBurst(false), 1600);
  }

  const pulsing = playing || burst;

  return (
    <button
      type="button"
      className={`play-sound${pulsing ? " is-playing" : ""}${className ? ` ${className}` : ""}`}
      aria-label="播放读音"
      aria-pressed={pulsing}
      onClick={handlePlay}
    >
      <span className="play-sound-rings" aria-hidden="true">
        <i />
        <i />
        <i />
      </span>
      <span className="play-sound-face" aria-hidden="true">
        <svg viewBox="0 0 24 24" fill="currentColor" data-icon="play" aria-hidden="true">
          <path d="M4.5 5.653c0-1.427 1.529-2.33 2.779-1.643l11.54 6.347c1.295.712 1.295 2.573 0 3.286L7.28 19.99c-1.25.687-2.779-.217-2.779-1.643V5.653Z" />
        </svg>
      </span>
    </button>
  );
}
