import { Link, NavLink } from "react-router";
import { Icon } from "@/components/icons";
import { Button } from "@/components/ui/button";
import { DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuLabel,
  DropdownMenuSeparator, DropdownMenuTrigger } from "@/components/ui/dropdown-menu";
import { useTheme } from "@/hooks/use-theme";
import { cn } from "@/lib/utils";

export function Header() {
  const { theme, toggleTheme } = useTheme();
  return <header className="flex flex-wrap items-center justify-between gap-4 px-page py-5 sm:py-7">
    <Link to="/" aria-label="SafeDrop home" className="flex items-center gap-2.5 rounded-md">
      <span className="grid size-10 place-items-center rounded-2xl bg-primary text-primary-foreground">
        <Icon name="drop" className="size-6" />
      </span>
      <span className="text-xl font-semibold tracking-tight">SafeDrop<span className="text-primary">.</span></span>
    </Link>
    <nav aria-label="Main navigation" className="order-3 flex w-full justify-center gap-1 sm:order-none sm:w-auto">
      {[{ to: "/", label: "Upload" }, { to: "/files", label: "My files" }].map(({ to, label }) =>
        <NavLink key={to} to={to} end className={({ isActive }) => cn(
          "inline-flex min-h-11 items-center rounded-full px-5 text-sm font-medium transition-colors",
          isActive ? "bg-secondary text-secondary-foreground" : "text-muted-foreground hover:bg-muted hover:text-foreground",
        )}>{label}</NavLink>,
      )}
    </nav>
    <div className="flex items-center gap-1.5">
      <Button variant="ghost" size="icon" onClick={toggleTheme}
        aria-label={`Switch to ${theme === "dark" ? "light" : "dark"} mode`}>
        <Icon name={theme === "dark" ? "sun" : "moon"} />
      </Button>
      <DropdownMenu>
        <DropdownMenuTrigger asChild>
          <Button variant="outline" className="gap-2 px-3" aria-label="User menu">
            <Icon name="user" /><Icon name="chevron" className="size-4" />
          </Button>
        </DropdownMenuTrigger>
        <DropdownMenuContent align="end">
          <DropdownMenuLabel className="px-3 py-2 text-xs text-muted-foreground">Not signed in</DropdownMenuLabel>
          <DropdownMenuSeparator />
          <DropdownMenuItem asChild><Link to="/login">Sign in</Link></DropdownMenuItem>
          <DropdownMenuItem asChild><Link to="/register">Create account</Link></DropdownMenuItem>
        </DropdownMenuContent>
      </DropdownMenu>
    </div>
  </header>;
}
