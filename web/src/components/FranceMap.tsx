import { useEffect, useMemo, useState } from "react";
import { MapContainer, TileLayer, GeoJSON } from "react-leaflet";
import type { Feature, GeoJsonObject } from "geojson";
import type { Layer, PathOptions } from "leaflet";
import { getMap, type GeoWinner } from "../lib/api";
import { pct, safeColor } from "../lib/format";
import { AsyncState } from "./ui";

const GEOJSON_URL =
  "https://raw.githubusercontent.com/gregoiredavid/france-geojson/master/departements-version-simplifiee.geojson";

export function FranceMap({ cycle, round }: { cycle: string; round: number }) {
  const election = `presidentielle-${cycle}-t${round}`;
  const [geo, setGeo] = useState<GeoJsonObject>();
  const [winners, setWinners] = useState<GeoWinner[]>([]);
  const [error, setError] = useState<string>();
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    let alive = true;
    setLoading(true);
    setError(undefined);
    Promise.all([
      geo ? Promise.resolve(geo) : fetch(GEOJSON_URL).then((r) => r.json()),
      getMap(election, "departement"),
    ])
      .then(([g, w]) => {
        if (!alive) return;
        setGeo(g);
        setWinners(w);
        setLoading(false);
      })
      .catch((e) => {
        if (!alive) return;
        setError(String(e));
        setLoading(false);
      });
    return () => {
      alive = false;
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [election]);

  const byCode = useMemo(() => {
    const m = new Map<string, GeoWinner>();
    winners.forEach((w) => m.set(w.geo_code, w));
    return m;
  }, [winners]);

  const legend = useMemo(() => {
    const m = new Map<string, string>();
    winners.forEach((w) => m.set(w.winner, safeColor(w.color)));
    return [...m.entries()];
  }, [winners]);

  const style = (feature?: Feature): PathOptions => {
    const code = (feature?.properties as { code?: string })?.code ?? "";
    const w = byCode.get(code);
    return {
      fillColor: w ? safeColor(w.color) : "#e6e6ec",
      fillOpacity: w ? 0.82 : 0.35,
      color: "#ffffff",
      weight: 0.8,
    };
  };

  const onEach = (feature: Feature, layer: Layer) => {
    const props = feature.properties as { code?: string; nom?: string };
    const code = props?.code ?? "";
    const w = byCode.get(code);
    const name = w?.geo_name || props?.nom || code;
    layer.bindTooltip(
      w
        ? `<b>${name}</b> <span style="color:#8a8a9a">(${code})</span><br/>${w.winner} — <b>${pct(
            w.pct
          )} %</b>`
        : `<b>${name}</b> (${code})<br/><i>pas de données</i>`,
      { className: "dept-tooltip", sticky: true }
    );
    layer.on({
      mouseover: (e) => (e.target as any).setStyle({ weight: 1.8, fillOpacity: 0.95 }),
      mouseout: (e) => (e.target as any).setStyle(style(feature)),
    });
  };

  const noData = !loading && !error && winners.length === 0;

  return (
    <div className="card">
      <div className="card-head">
        <div>
          <h3>Carte de France — candidat arrivé en tête par département</h3>
          <div className="meta">
            Présidentielle {cycle} · {round === 1 ? "1er tour" : "2nd tour"}
          </div>
        </div>
      </div>

      <AsyncState
        loading={loading}
        error={error}
        empty={noData}
        emptyText={`Pas de résultats cartographiques pour ${cycle}, tour ${round}.`}
      />

      {!loading && !error && winners.length > 0 && (
        <>
          <div className="map-wrap">
            <MapContainer
              center={[46.6, 2.4]}
              zoom={6}
              minZoom={5}
              scrollWheelZoom={false}
              style={{ height: "100%", width: "100%" }}
            >
              <TileLayer
                attribution='&copy; OpenStreetMap · CARTO'
                url="https://{s}.basemaps.cartocdn.com/light_all/{z}/{x}/{y}{r}.png"
              />
              {geo && (
                <GeoJSON
                  key={election}
                  data={geo}
                  style={style}
                  onEachFeature={onEach}
                />
              )}
            </MapContainer>
          </div>
          <div className="legend">
            {legend.map(([name, color]) => (
              <span className="item" key={name}>
                <i className="swatch" style={{ background: color }} /> {name}
              </span>
            ))}
          </div>
        </>
      )}
    </div>
  );
}
