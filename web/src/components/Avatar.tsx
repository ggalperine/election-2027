import { useState } from "react";
import { candidatePhoto, initials } from "../lib/candidatePhoto";

// Avatar renders a small round candidate portrait, falling back to the
// candidate's initials on a colored disc when no photo exists or the file
// fails to load.
export function Avatar({
  name,
  color,
  size = 40,
}: {
  name: string;
  color: string;
  size?: number;
}) {
  const src = candidatePhoto(name);
  const [failed, setFailed] = useState(false);

  if (!src || failed) {
    return (
      <span
        className="avatar avatar-fallback"
        style={{ width: size, height: size, background: color }}
        aria-hidden="true"
      >
        {initials(name)}
      </span>
    );
  }

  return (
    <img
      className="avatar"
      style={{ width: size, height: size }}
      src={src}
      alt=""
      loading="lazy"
      width={size}
      height={size}
      onError={() => setFailed(true)}
    />
  );
}
