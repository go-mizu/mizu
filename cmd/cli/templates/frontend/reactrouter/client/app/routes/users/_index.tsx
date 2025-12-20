import { Link, useLoaderData } from "react-router";
import type { Route } from "./+types/_index";

interface User {
  id: number;
  name: string;
  email: string;
  role: string;
}

export async function loader() {
  const res = await fetch("/api/users");
  const users: User[] = await res.json();
  return { users };
}

export function meta({}: Route.MetaArgs) {
  return [
    { title: "Users - {{.Name}}" },
    { name: "description", content: "View all users" },
  ];
}

export default function Users({ loaderData }: Route.ComponentProps) {
  const { users } = loaderData;

  return (
    <div className="users-list">
      <div className="users-grid">
        {users.map((user) => (
          <Link
            key={user.id}
            to={`/users/${user.id}`}
            className="user-card"
          >
            <div className="user-avatar">
              {user.name.charAt(0).toUpperCase()}
            </div>
            <div className="user-info">
              <h3>{user.name}</h3>
              <p>{user.email}</p>
              <span className={`badge badge-${user.role}`}>
                {user.role}
              </span>
            </div>
          </Link>
        ))}
      </div>
    </div>
  );
}
