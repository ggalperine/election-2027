export function MethodologyFooter() {
  return (
    <footer className="method-footer">
      <h4>Méthodologie</h4>
      <p>
        Les estimations reposent sur une <strong>moyenne pondérée de sondages</strong>{" "}
        (« poll of polls »). Chaque sondage est pondéré par la taille de son
        échantillon et par une <strong>décroissance temporelle</strong> qui donne
        plus de poids aux enquêtes récentes.
      </p>
      <p>
        L'<strong>intervalle de confiance à 95 %</strong> combine deux sources
        d'incertitude : l'erreur d'échantillonnage propre à chaque sondage et la{" "}
        <strong>dispersion entre instituts</strong> (marge et biais de maison),
        additionnées en quadrature (⊕).
      </p>
      <p>
        Les projections comparent deux modèles : le{" "}
        <strong>lissage exponentiel de Holt</strong> (<code>holt</code>, tendance
        adaptative) et une <strong>régression linéaire pondérée</strong>{" "}
        (<code>linreg</code>). La bande d'incertitude s'élargit avec l'horizon de
        prévision.
      </p>
      <p className="cred">
        Sources : les sondages sont recensés à partir de la{" "}
        <a
          href="https://www.commission-des-sondages.fr"
          target="_blank"
          rel="noopener noreferrer"
        >
          Commission des sondages
        </a>{" "}
        — l'autorité légale auprès de laquelle chaque institut (Ifop, Ipsos,
        OpinionWay, Harris Interactive, Elabe, Odoxa, Cluster17, Verian, …) dépose
        la notice officielle de toute enquête publiée. Le lien vers cette notice
        figure dans le tableau des sources. Résultats officiels par département :
        données ouvertes du ministère de l'Intérieur.
      </p>
    </footer>
  );
}
