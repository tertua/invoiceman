import { Skeleton } from "@/components/ui/Skeleton";
import { TableSkeleton } from "@/components/ui/TableSkeleton";

// Keeps the page-title block (admin pages render this before PageHeader);
// rows delegate to the shared table skeleton so admin tables match the app.
export function AdminTableSkeleton({ rows = 5, columns = 5, minWidthClassName = "min-w-[700px]" }) {
  return (
    <div>
      <div className="mb-6 space-y-2">
        <Skeleton className="h-7 w-48 rounded-xl" />
        <Skeleton className="h-4 w-72 rounded-xl" />
      </div>
      <TableSkeleton variant="table" rows={rows} columns={columns} minWidthClassName={minWidthClassName} />
    </div>
  );
}
