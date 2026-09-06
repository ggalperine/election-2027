import { useState } from "react";
import { postContact } from "../lib/api";

// Demand-capture form: validates B2B interest (custom analysis, datasets,
// canvassing targeting, bespoke dev) as leads BEFORE building. Stored via the
// same /api/contact backend, tagged in the subject/message.
const NEEDS = [
  "Analyse candidat approfondie",
  "Jeu de données / export (CSV, API)",
  "Ciblage électoral (porte-à-porte, bureaux de vote)",
  "Développement sur-mesure",
  "Autre",
];
const PROFILES = [
  "Média / presse",
  "Campagne / parti",
  "Recherche / université",
  "Entreprise",
  "Particulier",
  "Autre",
];

type Status = "idle" | "sending" | "ok" | "error";

export function RequestForm() {
  const [name, setName] = useState("");
  const [email, setEmail] = useState("");
  const [need, setNeed] = useState(NEEDS[0]);
  const [profile, setProfile] = useState(PROFILES[0]);
  const [details, setDetails] = useState("");
  const [website, setWebsite] = useState("");
  const [status, setStatus] = useState<Status>("idle");

  async function submit(e: React.FormEvent) {
    e.preventDefault();
    if (status === "sending") return;
    setStatus("sending");
    try {
      await postContact({
        name,
        email,
        subject: `[Demande sur-mesure] ${need}`,
        message: `Profil : ${profile}\nBesoin : ${need}\n\n${details}`,
        website,
      });
      setStatus("ok");
      setDetails("");
    } catch {
      setStatus("error");
    }
  }

  if (status === "ok") {
    return (
      <div className="card contact-card">
        <p className="contact-done">
          ✅ Demande reçue. Nous revenons vers vous pour en discuter.
        </p>
      </div>
    );
  }

  return (
    <div className="card contact-card">
      <form className="contact-form" onSubmit={submit}>
        <div className="contact-row">
          <label>
            Votre besoin
            <select value={need} onChange={(e) => setNeed(e.target.value)}>
              {NEEDS.map((n) => (
                <option key={n} value={n}>
                  {n}
                </option>
              ))}
            </select>
          </label>
          <label>
            Votre profil
            <select value={profile} onChange={(e) => setProfile(e.target.value)}>
              {PROFILES.map((p) => (
                <option key={p} value={p}>
                  {p}
                </option>
              ))}
            </select>
          </label>
        </div>
        <div className="contact-row">
          <label>
            Nom / organisation
            <input
              type="text"
              value={name}
              onChange={(e) => setName(e.target.value)}
              required
              maxLength={200}
            />
          </label>
          <label>
            Email
            <input
              type="email"
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              required
              maxLength={200}
              autoComplete="email"
            />
          </label>
        </div>
        <label>
          Détails
          <textarea
            value={details}
            onChange={(e) => setDetails(e.target.value)}
            required
            maxLength={5000}
            rows={4}
            placeholder="Décrivez la donnée, l'analyse ou le développement souhaité…"
          />
        </label>
        <input
          className="contact-hp"
          type="text"
          tabIndex={-1}
          autoComplete="off"
          value={website}
          onChange={(e) => setWebsite(e.target.value)}
          aria-hidden="true"
        />
        <div className="contact-actions">
          <button type="submit" disabled={status === "sending"}>
            {status === "sending" ? "Envoi…" : "Envoyer la demande"}
          </button>
          {status === "error" && (
            <span className="contact-err">
              Échec — réessayez ou écrivez à contact@elyseometre.fr.
            </span>
          )}
        </div>
      </form>
    </div>
  );
}
