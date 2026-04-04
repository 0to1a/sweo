import { rootRoute } from "./__root";
import { indexRoute } from "./index";
import { sessionRoute } from "./sessions.$sessionId";

export const routeTree = rootRoute.addChildren([indexRoute, sessionRoute]);
