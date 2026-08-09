import { BrandLogo } from "../shell/BrandLogo";

export function LoadingScreen() {
  return (
    <div className="vocat-boot-loader" role="status" aria-live="polite" aria-label="Loading VoCat">
      <BrandLogo className="vocat-boot-logo" />
      <span className="sr-only">Loading VoCat</span>
    </div>
  );
}
