import { useLoaderData, useNavigate } from "react-router";
import type { Route } from "./+types/$id";

interface User {
  id: string;
  name: string;
  email: string;
  role: string;
}

export async function loader({ params }: Route.LoaderArgs) {
  const res = await fetch(`/api/users/${params.id}`);

  if (!res.ok) {
    throw new Response("User not found", { status: 404 });
  }

  const user: User = await res.json();
  return { user };
}

export function meta({ data }: Route.MetaArgs) {
  return [
    { title: `${data?.user.name || "User"} - {{.Name}}` },
    { name: "description", content: `View ${data?.user.name}'s profile` },
  ];
}

export default function UserDetail({ loaderData }: Route.ComponentProps) {
  const { user } = loaderData;
  const navigate = useNavigate();

  return (
    <div className="user-detail">
      <button onClick={() => navigate(-1)} className="back-button">
        ← Back to Users
      </button>

      <div className="user-profile">
        <div className="user-avatar-large">
          {user.name.charAt(0).toUpperCase()}
        </div>

        <div className="user-details">
          <h2>{user.name}</h2>

          <div className="detail-row">
            <span className="label">Email:</span>
            <span className="value">{user.email}</span>
          </div>

          <div className="detail-row">
            <span className="label">Role:</span>
            <span className={`badge badge-${user.role}`}>
              {user.role}
            </span>
          </div>

          <div className="detail-row">
            <span className="label">ID:</span>
            <span className="value">{user.id}</span>
          </div>
        </div>
      </div>
    </div>
  );
}

export function ErrorBoundary({ error }: Route.ErrorBoundaryProps) {
  return (
    <div className="error-page">
      <h1>Oops!</h1>
      <p>
        {error instanceof Error
          ? error.message
          : "Something went wrong loading this user."}
      </p>
      <a href="/users" className="button">
        Back to Users
      </a>
    </div>
  );
}
