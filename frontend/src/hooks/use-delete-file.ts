import { useQueryClient, type InfiniteData } from "@tanstack/react-query";
import { deleteFile } from "@/api/files";
import type { FileItem, FilePage } from "@/api/schemas";
import { useToast } from "@/hooks/use-toast";

export function useDeleteFile() {
  const client = useQueryClient();
  const { toast } = useToast();
  return async (file: FileItem) => {
    await client.cancelQueries({ queryKey: ["files"] });
    const previous = client.getQueryData<InfiniteData<FilePage>>(["files"]);
    const restore = () => {
      client.setQueryData<InfiniteData<FilePage>>(["files"], (current) => {
        if (!current) return previous;
        if (current.pages.some((page) => page.items.some((item) => item.id === file.id))) return current;
        const index = previous?.pages.findIndex((page) => page.items.some((item) => item.id === file.id)) ?? 0;
        const pageIndex = Math.max(0, Math.min(index, current.pages.length - 1));
        return { ...current, pages: current.pages.map((page, i) => i === pageIndex
          ? { ...page, items: [...page.items, file].sort((a, b) => b.created_at.localeCompare(a.created_at)) } : page) };
      });
    };
    client.setQueryData<InfiniteData<FilePage>>(["files"], (current) => current && ({
      ...current, pages: current.pages.map((page) => ({ ...page, items: page.items.filter((item) => item.id !== file.id) })),
    }));
    let resolved = false;
    toast({
      title: "File removed", description: "Undo before this notification closes to keep your file.",
      action: { label: "Undo", onClick: () => { if (!resolved) { resolved = true; restore(); } } },
      onDismiss: () => {
        if (resolved) return;
        resolved = true;
        void deleteFile(file.id).then(() => {
          void client.invalidateQueries({ queryKey: ["usage"] });
        }).catch((error: unknown) => {
          restore();
          toast({ title: "Could not delete file", description: error instanceof Error ? error.message : "Please try again.", variant: "destructive" });
        });
      },
    });
  };
}
