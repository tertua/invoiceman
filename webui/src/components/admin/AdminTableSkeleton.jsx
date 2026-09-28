import { Skeleton } from "@/components/ui/Skeleton";

export function AdminTableSkeleton({ rows = 5 }) {
  return (
    <div>
      <div className="mb-6 space-y-2">
        <Skeleton className="h-7 w-48 rounded-xl" />
        <Skeleton className="h-4 w-72 rounded-xl" />
      </div>
      <div className="space-y-2">
        {Array.from({ length: rows }).map((_, i) => (
          <Skeleton key={i} className="h-16 rounded-2xl" />
        ))}
      </div>
    </div>
  );
}
