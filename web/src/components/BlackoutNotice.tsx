import { BlackoutWindow, formatParis } from "../lib/blackout";

export function BlackoutNotice({ window }: { window: BlackoutWindow }) {
  return (
    <section className="blackout" role="alert">
      <div className="blackout-mark">⚖️</div>
      <h2>Diffusion des sondages suspendue</h2>
      <p>
        Conformément à la <strong>loi n°77-808 du 19&nbsp;juillet 1977</strong>{" "}
        (article&nbsp;11), la publication et la diffusion de sondages électoraux
        sont <strong>interdites la veille et le jour du scrutin</strong>, jusqu'à
        la fermeture du dernier bureau de vote en métropole.
      </p>
      <p className="blackout-scrutin">Scrutin concerné : {window.label}.</p>
      <p className="blackout-until">
        Les estimations, moyennes pondérées et projections seront de nouveau
        accessibles le <strong>{formatParis(window.endUtc)}</strong>.
      </p>
      <p className="blackout-foot">
        Merci de votre compréhension. Les résultats officiels seront publiés par
        le ministère de l'Intérieur après la fermeture des bureaux de vote.
      </p>
    </section>
  );
}
