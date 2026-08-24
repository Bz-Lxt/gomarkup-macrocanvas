import { defineStore } from "pinia";
import { api, clearToken, getToken, setToken } from "../api/client";

export const useSession = defineStore("session", {
  state: () => ({
    token: getToken(),
    username: "" as string,
    toast: "" as string,
    toastKind: "ok" as "ok" | "err",
  }),
  actions: {
    async login(u: string, p: string) {
      const d = await api.login(u, p);
      this.token = d.token;
      this.username = d.username;
      setToken(d.token);
    },
    logout() {
      this.token = "";
      clearToken();
    },
    flash(msg: string, kind: "ok" | "err" = "ok") {
      this.toast = msg;
      this.toastKind = kind;
      setTimeout(() => {
        if (this.toast === msg) this.toast = "";
      }, 5000);
    },
  },
});
