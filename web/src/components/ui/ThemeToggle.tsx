import { useTheme } from "../../context/ThemeContext";
import { Button } from "./index";

export function ThemeToggle() {
  const { theme, toggle } = useTheme();
  return (
    <Button
      variant="default"
      className="btn-icon"
      onClick={toggle}
      title={theme === "light" ? "Modo oscuro" : "Modo claro"}
      aria-label="Cambiar tema"
    >
      {theme === "light" ? "🌙" : "☀️"}
    </Button>
  );
}
