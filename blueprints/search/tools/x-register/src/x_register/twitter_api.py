"""Twitter/X internal API client using curl_cffi for TLS fingerprinting.

All requests use Chrome impersonation to match real browser TLS signatures.
"""

from __future__ import annotations

import json
import re
import time
import urllib.parse
from dataclasses import dataclass, field
from typing import Any

from curl_cffi import requests as curl_requests

# Public bearer token — same across all known X clients
BEARER = (
    "AAAAAAAAAAAAAAAAAAAAANRILgAAAAAAnNwIzUejRCOuH5E6I8xnZz4puTs"
    "%3D1Zv7ttfk8LF81IUq16cHjhLTvJu4FA33AGWWjCpTnA"
)

# GraphQL tweet creation endpoint (stable)
CREATE_TWEET_URL = (
    "https://twitter.com/i/api/graphql/oB-5XsHNAbjvARJEc8CZFw/CreateTweet"
)

# Onboarding API base
API_BASE = "https://api.x.com/1.1"

# Subtask version map required by X onboarding API
SUBTASK_VERSIONS: dict[str, int] = {
    "generic_urt": 1,
    "standard": 1,
    "open_home_tweets": 1,
    "app_locale_update": 1,
    "enter_date_of_birth": 1,
    "email_verification": 2,
    "enter_password": 5,
    "enter_text": 5,
    "follow_by_id": 1,
    "js_instrumentation": 1,
    "menu_dialog": 1,
    "notifications_permission_prompt": 2,
    "open_account": 2,
    "open_link": 1,
    "open_external_link": 1,
    "phone_verification": 4,
    "privacy_options": 1,
    "security_key": 3,
    "select_avatar": 1,
    "select_banner": 2,
    "settings_list": 7,
    "show_code": 1,
    "sign_up": 2,
    "sign_up_review": 4,
    "tweet_selection_urt": 1,
    "update_users": 1,
    "upload_media": 1,
    "user_recommendations_list": 4,
    "user_recommendations_urt": 1,
    "wait_spinner": 3,
    "web_modal": 1,
}

# Elon's account — required follow during onboarding gating step
REQUIRED_FOLLOW_ID = "44196397"

# CreateTweet features map
TWEET_FEATURES: dict[str, bool] = {
    "tweetypie_unmention_optimization_enabled": True,
    "responsive_web_edit_tweet_api_enabled": True,
    "graphql_is_translatable_rweb_tweet_is_translatable_enabled": True,
    "view_counts_everywhere_api_enabled": True,
    "longform_notetweets_consumption_enabled": True,
    "tweet_awards_web_tipping_enabled": False,
    "longform_notetweets_rich_text_read_enabled": True,
    "longform_notetweets_inline_media_enabled": False,
    "responsive_web_graphql_exclude_directive_enabled": True,
    "verified_phone_label_enabled": False,
    "freedom_of_speech_not_reach_fetch_enabled": True,
    "standardized_nudges_misinfo": True,
    "tweet_with_visibility_results_prefer_gql_limited_actions_policy_enabled": False,
    "responsive_web_graphql_skip_user_profile_image_extensions_enabled": False,
    "responsive_web_graphql_timeline_navigation_enabled": True,
    "interactive_text_enabled": True,
    "responsive_web_text_conversations_enabled": False,
    "responsive_web_enhance_cards_enabled": False,
}


class TwitterError(Exception):
    pass


class PhoneVerificationRequired(Exception):
    """Raised when X requires phone verification — cannot automate."""


@dataclass
class SignupResult:
    auth_token: str
    ct0: str
    user_id: str
    screen_name: str


class TwitterSession:
    """Manages one X registration session via internal onboarding API."""

    def __init__(
        self,
        proxies: dict[str, str] | None = None,
        verbose: bool = False,
    ):
        self._proxies = proxies
        self._verbose = verbose
        self._session = curl_requests.Session(impersonate="chrome136")
        if proxies:
            self._session.proxies = proxies
        self._guest_token: str = ""
        self._ct0: str = ""

    def _log(self, msg: str) -> None:
        if self._verbose:
            ts = time.strftime("%H:%M:%S")
            print(f"[{ts}] [twitter] {msg}", flush=True)

    def _base_headers(self, authed: bool = False) -> dict[str, str]:
        h: dict[str, str] = {
            "authorization": f"Bearer {BEARER}",
            "content-type": "application/json",
            "x-twitter-active-user": "yes",
            "x-twitter-client-language": "en",
            "user-agent": (
                "Mozilla/5.0 (Windows NT 10.0; Win64; x64) "
                "AppleWebKit/537.36 (KHTML, like Gecko) "
                "Chrome/136.0.0.0 Safari/537.36"
            ),
            "origin": "https://x.com",
            "referer": "https://x.com/",
        }
        if self._guest_token and not authed:
            h["x-guest-token"] = self._guest_token
        if self._ct0:
            h["x-csrf-token"] = self._ct0
        return h

    def _post(self, url: str, body: Any, authed: bool = False) -> dict:
        self._log(f"POST {url}")
        resp = self._session.post(
            url,
            json=body,
            headers=self._base_headers(authed),
            timeout=30,
        )
        self._log(f"  -> {resp.status_code}")
        if resp.status_code == 429:
            raise TwitterError("Rate limited (429)")
        if resp.status_code >= 400:
            raise TwitterError(f"HTTP {resp.status_code}: {resp.text[:300]}")
        # Update ct0 from cookies
        ct0 = self._session.cookies.get("ct0")
        if ct0:
            self._ct0 = ct0
        return resp.json()

    # ------------------------------------------------------------------
    # Step 1: Guest token
    # ------------------------------------------------------------------

    def activate(self) -> str:
        """POST /guest/activate.json → guest_token.

        Always uses a direct (no-proxy) connection — guest token endpoint
        is not rate-limited by IP and proxies frequently time out here.
        """
        self._log("activating guest token (direct, no proxy)...")
        direct = curl_requests.Session(impersonate="chrome136")
        try:
            resp = direct.post(
                f"{API_BASE}/guest/activate.json",
                headers={
                    "authorization": f"Bearer {BEARER}",
                    "user-agent": self._base_headers()["user-agent"],
                },
                timeout=15,
            )
            if resp.status_code != 200:
                raise TwitterError(f"guest activate failed: {resp.status_code} {resp.text[:200]}")
            self._guest_token = resp.json()["guest_token"]
            # Copy guest_token cookie into main session
            self._session.cookies.set("gt", self._guest_token)
            self._log(f"guest_token={self._guest_token[:12]}...")
            return self._guest_token
        finally:
            direct.close()

    # ------------------------------------------------------------------
    # Step 2: Start signup flow
    # ------------------------------------------------------------------

    def start_signup(self) -> tuple[str, str | None, str | None]:
        """POST task.json?flow_name=signup → (flow_token, arkose_blob, arkose_link).

        Returns:
            flow_token: opaque flow state token
            arkose_blob: base64 blob from ArkoseEmail subtask URL (may be None)
            arkose_link: link name for ArkoseEmail subtask (may be None)
        """
        body = {
            "flow_name": "signup",
            "input_flow_data": {
                "flow_context": {
                    "debug_overrides": {},
                    "start_location": {"location": "splash_screen"},
                }
            },
            "subtask_versions": SUBTASK_VERSIONS,
        }
        data = self._post(f"{API_BASE}/onboarding/task.json?flow_name=signup", body)
        flow_token = data.get("flow_token", "")
        if not flow_token:
            raise TwitterError(f"No flow_token in signup response: {json.dumps(data)[:300]}")

        arkose_blob, arkose_link = self._extract_arkose(data)
        self._log(f"flow_token={flow_token[:20]}... arkose={arkose_blob is not None}")
        return flow_token, arkose_blob, arkose_link

    def _extract_arkose(self, data: dict) -> tuple[str | None, str | None]:
        """Extract Arkose blob and link from onboarding task response subtasks."""
        for subtask in data.get("subtasks", []):
            if subtask.get("subtask_id") == "ArkoseEmail":
                web_modal = subtask.get("web_modal", {})
                url = web_modal.get("url", "")
                link = web_modal.get("link", "")
                # Extract data= parameter from the Arkose URL
                parsed = urllib.parse.urlparse(url)
                params = urllib.parse.parse_qs(parsed.query)
                blob_list = params.get("data", [])
                if blob_list:
                    return blob_list[0], link
        return None, None

    # ------------------------------------------------------------------
    # Step 3: Begin email verification (triggers OTP send)
    # ------------------------------------------------------------------

    def begin_email_verification(
        self,
        flow_token: str,
        email: str,
        display_name: str,
        castle_token: str = "",
    ) -> str:
        """POST begin_verification.json → updated flow_token."""
        body = {
            "flow_token": flow_token,
            "email": email,
            "display_name": display_name,
        }
        if castle_token:
            body["castle_token"] = castle_token

        data = self._post(f"{API_BASE}/onboarding/begin_verification.json", body)
        new_token = data.get("flow_token", flow_token)
        self._log(f"begin_verification done, flow_token={new_token[:20]}...")
        return new_token

    # ------------------------------------------------------------------
    # Step 4: Submit Signup + EmailVerification subtasks
    # ------------------------------------------------------------------

    def submit_signup_and_otp(
        self,
        flow_token: str,
        *,
        js_instrumentation: str,
        display_name: str,
        email: str,
        birth_year: int,
        birth_month: int,
        birth_day: int,
        otp_code: str,
        arkose_token: str = "",
        arkose_link: str = "",
    ) -> tuple[str, str | None, str | None]:
        """Submit Signup subtask + EmailVerification subtask.

        Returns (flow_token, arkose_blob, arkose_link) from the response.
        """
        subtask_inputs: list[dict] = [
            {
                "subtask_id": "Signup",
                "sign_up": {
                    "js_instrumentation": {"response": js_instrumentation},
                    "link": "email_next_link",
                    "name": display_name,
                    "email": email,
                    "birthday": {
                        "day": birth_day,
                        "month": birth_month,
                        "year": birth_year,
                    },
                    "personalization_settings": {
                        "allow_cookie_use": False,
                        "allow_device_personalization": False,
                        "allow_partnerships": False,
                        "allow_ads_personalization": False,
                    },
                },
            },
        ]

        if arkose_token and arkose_link:
            subtask_inputs.append({
                "subtask_id": "ArkoseEmail",
                "web_modal": {
                    "completion_deeplink": (
                        f"twitter://onboarding/web_modal/next_link?access_token={arkose_token}"
                    ),
                    "link": arkose_link,
                },
            })

        subtask_inputs.append({
            "subtask_id": "EmailVerification",
            "email_verification": {
                "code": otp_code,
                "email": email,
                "link": "next_link",
            },
        })

        body = {
            "flow_token": flow_token,
            "subtask_inputs": subtask_inputs,
        }
        data = self._post(f"{API_BASE}/onboarding/task.json", body)
        new_token = data.get("flow_token", flow_token)
        arkose_blob, new_arkose_link = self._extract_arkose(data)
        self._log(f"signup+otp done, flow_token={new_token[:20]}...")
        return new_token, arkose_blob, new_arkose_link

    # ------------------------------------------------------------------
    # Step 5: Set password
    # ------------------------------------------------------------------

    def set_password(self, flow_token: str, password: str) -> str:
        """Submit EnterPassword subtask → flow_token."""
        body = {
            "flow_token": flow_token,
            "subtask_inputs": [
                {
                    "subtask_id": "EnterPassword",
                    "enter_password": {"password": password, "link": "next_link"},
                }
            ],
        }
        data = self._post(f"{API_BASE}/onboarding/task.json", body)
        token = data.get("flow_token", flow_token)
        self._log(f"set_password done")
        return token

    # ------------------------------------------------------------------
    # Steps 6+: Skip optional steps, complete onboarding
    # ------------------------------------------------------------------

    def _submit_skip(self, flow_token: str, subtask_id: str, link: str = "next_link") -> str:
        body = {
            "flow_token": flow_token,
            "subtask_inputs": [
                {"subtask_id": subtask_id, "link": link}
            ],
        }
        try:
            data = self._post(f"{API_BASE}/onboarding/task.json", body)
            return data.get("flow_token", flow_token)
        except TwitterError:
            return flow_token  # non-fatal skip

    def complete_onboarding(self, flow_token: str) -> SignupResult:
        """Drive through remaining optional onboarding subtasks until auth_token appears.

        Returns SignupResult with auth_token, ct0, user_id, screen_name.
        """
        MAX_STEPS = 15
        for step in range(MAX_STEPS):
            self._log(f"onboarding step {step}, flow_token={flow_token[:20]}...")

            # Check if auth_token cookie is already set
            auth_token = self._session.cookies.get("auth_token")
            if auth_token:
                ct0 = self._session.cookies.get("ct0", self._ct0)
                user_id, screen_name = self._extract_user_from_cookies()
                self._log(f"auth_token obtained! user_id={user_id} screen_name={screen_name}")
                return SignupResult(
                    auth_token=auth_token,
                    ct0=ct0,
                    user_id=user_id,
                    screen_name=screen_name,
                )

            body = {
                "flow_token": flow_token,
                "subtask_inputs": [],  # empty = "skip / proceed"
            }
            try:
                resp = self._session.post(
                    f"{API_BASE}/onboarding/task.json",
                    json=body,
                    headers=self._base_headers(),
                    timeout=30,
                )
            except Exception as e:
                raise TwitterError(f"onboarding step {step} failed: {e}") from e

            # Update ct0
            ct0_c = self._session.cookies.get("ct0")
            if ct0_c:
                self._ct0 = ct0_c

            if resp.status_code == 429:
                raise TwitterError("Rate limited during onboarding")

            auth_token = self._session.cookies.get("auth_token")
            if auth_token:
                ct0 = self._session.cookies.get("ct0", self._ct0)
                user_id, screen_name = self._extract_user_from_cookies()
                self._log(f"auth_token from cookie! user_id={user_id}")
                return SignupResult(
                    auth_token=auth_token,
                    ct0=ct0,
                    user_id=user_id,
                    screen_name=screen_name,
                )

            if resp.status_code >= 400:
                self._log(f"  step {step} error: {resp.status_code} {resp.text[:200]}")
                break

            data = resp.json()
            flow_token = data.get("flow_token", flow_token)

            subtasks = data.get("subtasks", [])
            self._log(f"  subtasks: {[s.get('subtask_id') for s in subtasks]}")

            for subtask in subtasks:
                sid = subtask.get("subtask_id", "")
                if sid == "PhoneVerification":
                    raise PhoneVerificationRequired(
                        "X requires phone verification — not automatable without SMS service"
                    )
                elif sid == "UserRecommendationsURTFollowGating":
                    # Must follow at least one account to proceed
                    flow_token = self._follow_and_proceed(flow_token, subtask)
                elif sid in ("SelectAvatar", "SelectBanner", "UsernameEntryBio",
                              "NotificationsPermissionPrompt",
                              "StandAloneCategoryPickerBlockingNextURT"):
                    # Skip optional steps
                    self._log(f"  skipping {sid}")

        # Last attempt to find auth_token
        auth_token = self._session.cookies.get("auth_token")
        if auth_token:
            ct0 = self._session.cookies.get("ct0", self._ct0)
            user_id, screen_name = self._extract_user_from_cookies()
            return SignupResult(
                auth_token=auth_token,
                ct0=ct0,
                user_id=user_id,
                screen_name=screen_name,
            )

        raise TwitterError("auth_token never appeared in cookies after onboarding")

    def _follow_and_proceed(self, flow_token: str, subtask: dict) -> str:
        """Follow required account (Elon) and proceed past the gating step."""
        self._log(f"  following {REQUIRED_FOLLOW_ID} to pass gating...")
        body = {
            "flow_token": flow_token,
            "subtask_inputs": [
                {
                    "subtask_id": "UserRecommendationsURTFollowGating",
                    "user_recommendations_urt": {
                        "users_to_follow": [REQUIRED_FOLLOW_ID],
                        "link": "next_link",
                    },
                }
            ],
        }
        try:
            data = self._post(f"{API_BASE}/onboarding/task.json", body)
            return data.get("flow_token", flow_token)
        except TwitterError:
            return flow_token

    def _extract_user_from_cookies(self) -> tuple[str, str]:
        """Try to extract user_id and screen_name from cookies / session."""
        # These may not be in cookies — return placeholders, caller can fetch profile
        user_id = self._session.cookies.get("twid", "").replace("u%3D", "").replace("u=", "")
        screen_name = ""
        return user_id, screen_name

    # ------------------------------------------------------------------
    # Tweet creation
    # ------------------------------------------------------------------

    def create_tweet(self, auth_token: str, ct0: str, text: str) -> str:
        """Post a tweet as the authenticated user. Returns tweet_id."""
        self._log(f"creating tweet: {text!r}")

        # Build an authenticated session with the new cookies
        tweet_session = curl_requests.Session(impersonate="chrome136")
        if self._proxies:
            tweet_session.proxies = self._proxies

        tweet_session.cookies.set("auth_token", auth_token, domain=".twitter.com")
        tweet_session.cookies.set("ct0", ct0, domain=".twitter.com")
        # Also set for x.com
        tweet_session.cookies.set("auth_token", auth_token, domain=".x.com")
        tweet_session.cookies.set("ct0", ct0, domain=".x.com")

        variables = {
            "tweet_text": text,
            "dark_request": False,
            "media": {"media_entities": [], "possibly_sensitive": False},
            "semantic_annotation_ids": [],
        }
        headers = {
            "authorization": f"Bearer {BEARER}",
            "content-type": "application/json",
            "x-twitter-active-user": "yes",
            "x-twitter-auth-type": "OAuth2Session",
            "x-csrf-token": ct0,
            "user-agent": self._base_headers()["user-agent"],
            "origin": "https://twitter.com",
            "referer": "https://twitter.com/",
        }

        resp = tweet_session.post(
            CREATE_TWEET_URL,
            json={"variables": variables, "features": TWEET_FEATURES},
            headers=headers,
            timeout=30,
        )
        self._log(f"  tweet response: {resp.status_code}")
        if resp.status_code >= 400:
            raise TwitterError(f"CreateTweet failed: {resp.status_code} {resp.text[:300]}")

        data = resp.json()
        try:
            tweet_id = (
                data["data"]["create_tweet"]["tweet_results"]["result"]["rest_id"]
            )
            self._log(f"  tweet_id={tweet_id}")
            return tweet_id
        except (KeyError, TypeError) as e:
            # Try to extract from response
            m = re.search(r'"rest_id":"(\d+)"', resp.text)
            if m:
                return m.group(1)
            raise TwitterError(f"Could not extract tweet_id from response: {e}") from e

    def close(self) -> None:
        self._session.close()
