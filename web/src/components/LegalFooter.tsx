export function LegalFooter() {
  return (
    <footer className="legal-footer">
      <div className="legal-grid">
        <section>
          <h4>Mentions légales</h4>
          <p>
            <strong>Éditeur :</strong> ESG Intelligence Consulting (ESGIC).
            Contact : <a href="mailto:contact@elyseometre.fr">contact@elyseometre.fr</a>.
          </p>
          <p>
            <strong>Hébergeur :</strong> Hetzner Online GmbH, Industriestr. 25,
            91710 Gunzenhausen, Allemagne — <a href="https://www.hetzner.com" target="_blank" rel="noopener noreferrer">hetzner.com</a>.
          </p>
          <p>
            <strong>Directeur de la publication :</strong> ESGIC.
          </p>
        </section>

        <section>
          <h4>Données &amp; vie privée</h4>
          <p>
            Ce site <strong>ne dépose aucun cookie</strong> et n'utilise
            <strong> aucun traceur publicitaire</strong>. Aucun compte n'est requis.
            La fréquentation est mesurée de façon <strong>anonyme et agrégée</strong> :
            un décompte de visites via une empreinte quotidienne à sens unique
            (IP + navigateur), <strong>sans cookie</strong>, non réversible et non
            reliable d'un jour à l'autre. Aucune donnée personnelle n'est conservée
            ni revendue.
          </p>
          <p>
            Conforme au <strong>RGPD</strong> : mesure d'audience anonyme exemptée
            de consentement (délibération CNIL). Les seules données affichées sont
            des <strong>sondages publics agrégés</strong> et des{" "}
            <strong>résultats électoraux officiels</strong>.
          </p>
        </section>

        <section>
          <h4>Avertissement</h4>
          <p>
            Elyséomètre est un service <strong>indépendant</strong> d'agrégation de
            sondages, à but <strong>informatif et éducatif</strong>. Il n'est
            affilié à aucun parti, candidat, institut de sondage ni autorité
            publique.
          </p>
          <p>
            Les estimations et projections sont des <strong>modèles
            statistiques</strong>, comportent une marge d'incertitude et
            <strong> ne constituent en rien une prédiction</strong> du résultat. Elles
            ne sauraient engager la responsabilité de l'éditeur.
          </p>
        </section>
      </div>

      <p className="legal-copy">
        © 2026 Elyséomètre — elyseometre.fr. Tous droits réservés. Les données
        sources restent la propriété de leurs détenteurs respectifs.
      </p>
    </footer>
  );
}
