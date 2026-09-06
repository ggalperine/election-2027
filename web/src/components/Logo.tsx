export function Logo({ size = 34 }: { size?: number }) {
  return (
    <svg
      width={size}
      height={size}
      viewBox="0 0 48 48"
      role="img"
      aria-label="Elyséomètre"
      fill="none"
      xmlns="http://www.w3.org/2000/svg"
    >
      {/* Dôme de l'Élysée */}
      <path
        d="M8 30c0-8.837 7.163-16 16-16s16 7.163 16 16"
        stroke="currentColor"
        strokeWidth="2.5"
        strokeLinecap="round"
      />
      <line x1="24" y1="7" x2="24" y2="12" stroke="currentColor" strokeWidth="2.5" strokeLinecap="round" />
      <circle cx="24" cy="6" r="1.8" fill="currentColor" />
      {/* Barres de sondage tricolores */}
      <rect x="14" y="30" width="5.5" height="10" rx="1.2" fill="#0055A4" />
      <rect x="21.25" y="24" width="5.5" height="16" rx="1.2" fill="currentColor" />
      <rect x="28.5" y="27" width="5.5" height="13" rx="1.2" fill="#EF4135" />
      {/* Socle */}
      <line x1="10" y1="41" x2="38" y2="41" stroke="currentColor" strokeWidth="2.5" strokeLinecap="round" />
    </svg>
  );
}
