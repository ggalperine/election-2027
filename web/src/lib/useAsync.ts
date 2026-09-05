import { useEffect, useState } from "react";

export interface AsyncState<T> {
  data: T | undefined;
  error: string | undefined;
  loading: boolean;
}

/**
 * Runs an async factory whenever `deps` change, tracking loading / error state.
 * Guards against setting state after unmount / stale requests.
 */
export function useAsync<T>(
  factory: () => Promise<T>,
  deps: unknown[]
): AsyncState<T> {
  const [state, setState] = useState<AsyncState<T>>({
    data: undefined,
    error: undefined,
    loading: true,
  });

  useEffect(() => {
    let alive = true;
    setState((s) => ({ ...s, loading: true, error: undefined }));
    factory()
      .then((data) => {
        if (alive) setState({ data, error: undefined, loading: false });
      })
      .catch((e) => {
        if (alive)
          setState({ data: undefined, error: String(e), loading: false });
      });
    return () => {
      alive = false;
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, deps);

  return state;
}
