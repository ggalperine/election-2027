import { useMemo, useState } from "react";
import { getCycles } from "./lib/api";
import { useAsync } from "./lib/useAsync";
import { Section, Segmented } from "./components/ui";
import { Summary } from "./components/Summary";
import { Momentum } from "./components/Momentum";
import { Leaderboard } from "./components/Leaderboard";
import { TrendTracker } from "./components/TrendTracker";
import { Forecast } from "./components/Forecast";
import { ScenarioLab } from "./components/ScenarioLab";
import { PollVsOutcome } from "./components/PollVsOutcome";
import { Instituts } from "./components/Instituts";
import { LatestPolls } from "./components/LatestPolls";
import { SourcesTable } from "./components/SourcesTable";
// FranceMap kept on disk but no longer rendered.
// import { FranceMap } from "./components/FranceMap";
import { MethodologyFooter } from "./components/MethodologyFooter";
import { LegalFooter } from "./components/LegalFooter";
import { Logo } from "./components/Logo";
import { BlackoutNotice } from "./components/BlackoutNotice";
import { activeBlackout } from "./lib/blackout";
import { Probabilities } from "./components/Probabilities";

const WINDOW_OPTIONS = [
  { value: "14", label: "Récent" },
  { value: "30", label: "30 j" },
  { value: "90", label: "90 j" },
  { value: "0", label: "Depuis le début" },
] as const;

export function App() {
  const { data: cycles } = useAsync(() => getCycles(), []);
  const [cycle, setCycle] = useState("2027");
  const [round, setRound] = useState<1 | 2>(1);
  const [window, setWindow] = useState(30); // "moyenne glissante 30j" (default)

  const cycleList = cycles ?? [
    { cycle: "2027", has_actual: false },
    { cycle: "2022", has_actual: true },
  ];

  const hasActual = useMemo(
    () => cycleList.find((c) => c.cycle === cycle)?.has_actual ?? false,
    [cycleList, cycle]
  );

  const blackout = activeBlackout();

  return (
    <>
      <header className="site-header">
        <div className="inner">
          <div className="brand">
            <span className="brand-logo"><Logo size={34} /></span>
            <span className="brand-name">Elyséomètre</span>
          </div>
          <p className="eyebrow">Agrégateur de sondages · France</p>
          <h1>Présidentielle française — le tracker</h1>
          <p className="lede">
            Moyenne pondérée des sondages, intervalles de confiance à 95 % et
            projections. Une lecture scientifique et transparente de la course
            présidentielle.
          </p>
        </div>
      </header>

      {blackout ? (
        <main className="app">
          <BlackoutNotice window={blackout} />
          <LegalFooter />
        </main>
      ) : (
       <>
      <nav className="controls" aria-label="Filtres globaux">
        <div className="inner">
          <div className="control-group">
            <span className="control-label">Cycle</span>
            <Segmented
              value={cycle}
              onChange={setCycle}
              options={cycleList.map((c) => ({
                value: c.cycle,
                label: c.cycle,
              }))}
            />
          </div>
          <div className="control-group">
            <span className="control-label">Tour</span>
            <Segmented<"1" | "2">
              value={String(round) as "1" | "2"}
              onChange={(v) => setRound(v === "1" ? 1 : 2)}
              options={[
                { value: "1", label: "1er tour" },
                { value: "2", label: "2nd tour" },
              ]}
            />
          </div>
          <div className="control-group">
            <span className="control-label">Période</span>
            <Segmented
              value={String(window)}
              onChange={(v) => setWindow(Number(v))}
              options={WINDOW_OPTIONS.map((o) => ({
                value: o.value,
                label: o.label,
              }))}
            />
          </div>
        </div>
      </nav>

      <main className="app">
        <Summary cycle={cycle} round={round} window={window} />

        <Section
          kicker="Instantané"
          title="Où en est la course"
          sub="Intentions de vote actuelles, agrégées et pondérées, avec leur marge d'incertitude."
        >
          <Leaderboard cycle={cycle} round={round} window={window} />
        </Section>

        <Section
          kicker="Probabilités"
          title="Quelles chances de qualification et de victoire"
          sub="Simulation Monte Carlo à partir de la distribution prédictive des intentions de vote. Probabilité d'accéder au 2nd tour, d'arriver en tête et de l'emporter."
        >
          <Probabilities cycle={cycle} window={window} />
        </Section>

        <Section
          kicker="Dynamique"
          title="Qui progresse, qui recule"
          sub="Variation de la moyenne pondérée sur 30 jours."
        >
          <Momentum cycle={cycle} round={round} />
        </Section>

        <Section
          kicker="Tendance"
          title="L'évolution dans le temps"
          sub="Moyenne glissante par candidat. Activez la bande pour visualiser l'intervalle de confiance à 95 %."
        >
          <TrendTracker
            cycle={cycle}
            round={round}
            hasActual={hasActual}
            window={window}
          />
        </Section>

        <Section
          kicker="Prévision"
          title="Projection à l'horizon du scrutin"
          sub="Deux modèles au choix — lissage exponentiel de Holt ou régression linéaire pondérée. L'incertitude croît avec l'horizon."
        >
          <Forecast cycle={cycle} round={round} window={window} />
        </Section>

        <Section
          kicker="Scénarios"
          title="Et si… report des voix"
          sub="Certains candidats ne se présenteront pas tous — le bloc central alignera Attal ou Philippe, pas les deux. Modélisez un retrait et le transfert des voix à un allié."
        >
          <ScenarioLab cycle={cycle} round={round} window={window} />
        </Section>

        {hasActual && (
          <Section
            kicker="Vérification"
            title="Ce que disaient les sondages vs le résultat"
            sub="Comparaison de la moyenne finale des sondages avec le résultat officiel du scrutin."
          >
            <PollVsOutcome cycle={cycle} round={round} />
          </Section>
        )}

        <Section
          kicker="Instituts"
          title="Les instituts"
          sub="Qui sonde, à quelle fréquence, et comment chaque maison s'écarte de la moyenne de tous les sondages."
        >
          <Instituts cycle={cycle} round={round} />
        </Section>

        <Section
          kicker="Actualité"
          title="Derniers sondages"
          sub="Les enquêtes les plus récentes prises en compte dans l'agrégation."
        >
          <LatestPolls cycle={cycle} round={round} />
        </Section>

        <Section
          kicker="Transparence"
          title="Toutes les sources"
          sub="Chaque sondage individuel, son institut, son commanditaire et le lien vers la notice officielle."
        >
          <SourcesTable cycle={cycle} round={round} />
        </Section>

        <MethodologyFooter />
        <LegalFooter />
      </main>
       </>
      )}
    </>
  );
}
