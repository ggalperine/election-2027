import { useState } from "react";
import { postContact } from "../lib/api";

type Status = "idle" | "sending" | "ok" | "error";

export function ContactForm() {
  const [name, setName] = useState("");
  const [email, setEmail] = useState("");
  const [subject, setSubject] = useState("");
  const [message, setMessage] = useState("");
  const [website, setWebsite] = useState(""); // honeypot
  const [status, setStatus] = useState<Status>("idle");

  async function submit(e: React.FormEvent) {
    e.preventDefault();
    if (status === "sending") return;
    setStatus("sending");
    try {
      await postContact({ name, email, subject, message, website });
      setStatus("ok");
      setName("");
      setEmail("");
      setSubject("");
      setMessage("");
    } catch {
      setStatus("error");
    }
  }

  if (status === "ok") {
    return (
      <div className="card contact-card">
        <p className="contact-done">
          ✅ Message envoyé. Merci — nous revenons vers vous dès que possible.
        </p>
      </div>
    );
  }

  return (
    <div className="card contact-card">
      <form className="contact-form" onSubmit={submit}>
        <div className="contact-row">
          <label>
            Nom
            <input
              type="text"
              value={name}
              onChange={(e) => setName(e.target.value)}
              required
              maxLength={200}
              autoComplete="name"
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
          Sujet
          <input
            type="text"
            value={subject}
            onChange={(e) => setSubject(e.target.value)}
            maxLength={300}
          />
        </label>
        <label>
          Message
          <textarea
            value={message}
            onChange={(e) => setMessage(e.target.value)}
            required
            maxLength={5000}
            rows={5}
          />
        </label>
        {/* Honeypot: hidden from humans, bots fill it. */}
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
            {status === "sending" ? "Envoi…" : "Envoyer"}
          </button>
          {status === "error" && (
            <span className="contact-err">
              Échec de l'envoi. Réessayez ou écrivez à contact@elyseometre.fr.
            </span>
          )}
        </div>
      </form>
    </div>
  );
}
