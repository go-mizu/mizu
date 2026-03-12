"""SMS verification service clients.

Google requires a phone number for Gmail signup. This module supports:
  - smspool     smspool.net           (REST API, service ID 395 = Google/Gmail)
  - 5sim        5sim.net              (alternative, competitive pricing)
  - manual      user provides phone; OTP entered interactively

Usage:
  client = SmsPoolClient(api_key="...", verbose=True)
  number = client.get_number(country="us", service="google")
  otp = client.wait_for_otp(number.activation_id, timeout=120)
  client.finish(number.activation_id)
"""

from __future__ import annotations

import time
from dataclasses import dataclass

import httpx

POLL_INTERVAL = 5
POLL_TIMEOUT = 120


@dataclass
class PhoneNumber:
    number: str           # E.164 without +, e.g. "14155552671"
    activation_id: str    # service-specific ID for polling/canceling
    service: str          # "smspool" | "5sim" | "manual"


class SmsError(Exception):
    pass


class NoNumberAvailable(SmsError):
    pass


# ---------------------------------------------------------------------------
# smspool.net
# ---------------------------------------------------------------------------

class SmsPoolClient:
    """smspool.net REST API client.

    API docs: https://smspool.net/article/api
    Google/Gmail service ID: 395
    """
    BASE = "https://api.smspool.net"

    # Country IDs (full list via GET /country/retrieve_all)
    COUNTRIES = {"any": 1, "us": 1, "uk": 2, "ru": 7, "in": 15, "de": 24, "fr": 23}

    # Service IDs
    SERVICE_GOOGLE = "395"

    def __init__(self, api_key: str, verbose: bool = False):
        self._key = api_key
        self._verbose = verbose
        self._client = httpx.Client(timeout=20)

    def _log(self, msg: str) -> None:
        if self._verbose:
            ts = time.strftime("%H:%M:%S")
            print(f"[{ts}] [smspool] {msg}", flush=True)

    def _post(self, path: str, data: dict) -> dict:
        data["key"] = self._key
        resp = self._client.post(f"{self.BASE}{path}", data=data)
        resp.raise_for_status()
        return resp.json()

    def get_balance(self) -> float:
        data = self._post("/request/balance", {})
        return float(data.get("balance", 0))

    def get_number(
        self,
        service: str = "google",
        country: str = "any",
    ) -> PhoneNumber:
        """Purchase a virtual number. Raises NoNumberAvailable if none found."""
        country_id = self.COUNTRIES.get(country, 1)
        service_id = self.SERVICE_GOOGLE  # always Google for Gmail
        self._log(f"purchasing number service={service_id} country={country} ({country_id})")
        data = self._post("/purchase/sms", {
            "country": country_id,
            "service": service_id,
        })
        self._log(f"response: {data}")
        if "order_id" in data:
            phone = data.get("phonenumber", "").lstrip("+")
            act_id = str(data["order_id"])
            self._log(f"got number: {phone} (id={act_id})")
            return PhoneNumber(number=phone, activation_id=act_id, service="smspool")
        if data.get("success") == 0:
            msg = data.get("message", str(data))
            if "stock" in msg.lower() or "balance" in msg.lower():
                raise NoNumberAvailable(msg)
            raise SmsError(f"purchase failed: {msg}")
        raise SmsError(f"unexpected response: {data}")

    def wait_for_otp(self, activation_id: str, timeout: int = POLL_TIMEOUT) -> str:
        """Poll for OTP. Returns 6-digit code."""
        self._log(f"polling for OTP (id={activation_id})...")
        deadline = time.time() + timeout
        while time.time() < deadline:
            time.sleep(POLL_INTERVAL)
            data = self._post("/sms/check", {"orderid": activation_id})
            self._log(f"  poll: {data}")
            sms = data.get("sms", "")
            if sms:
                import re
                m = re.search(r"\b(\d{6})\b", sms)
                if m:
                    code = m.group(1)
                    self._log(f"  OTP: {code}")
                    return code
            status = str(data.get("status", ""))
            if status in ("3", "CANCELLED", "EXPIRED"):
                raise SmsError(f"Activation {status}: {data}")
        raise SmsError(f"OTP not received within {timeout}s")

    def finish(self, activation_id: str) -> None:
        """Mark activation as successfully used."""
        self._post("/sms/set_status", {"orderid": activation_id, "status": "6"})
        self._log(f"finished activation {activation_id}")

    def cancel(self, activation_id: str) -> None:
        """Cancel unused activation."""
        self._post("/sms/cancel", {"orderid": activation_id})
        self._log(f"cancelled activation {activation_id}")

    def close(self) -> None:
        self._client.close()


# ---------------------------------------------------------------------------
# 5sim.net
# ---------------------------------------------------------------------------

class FiveSimClient:
    BASE = "https://5sim.net/v1"

    def __init__(self, api_key: str, verbose: bool = False):
        self._key = api_key
        self._verbose = verbose
        self._client = httpx.Client(
            timeout=15,
            headers={"Authorization": f"Bearer {api_key}", "Accept": "application/json"},
        )

    def _log(self, msg: str) -> None:
        if self._verbose:
            ts = time.strftime("%H:%M:%S")
            print(f"[{ts}] [5sim] {msg}", flush=True)

    def get_balance(self) -> float:
        resp = self._client.get(f"{self.BASE}/user/profile")
        resp.raise_for_status()
        return float(resp.json().get("balance", 0))

    def get_number(
        self,
        service: str = "google",
        country: str = "any",
    ) -> PhoneNumber:
        self._log(f"requesting {service} number country={country}")
        resp = self._client.get(
            f"{self.BASE}/user/buy/activation/{country}/any/{service}"
        )
        self._log(f"status: {resp.status_code}")
        if resp.status_code == 200:
            data = resp.json()
            phone = data["phone"].lstrip("+")
            act_id = str(data["id"])
            self._log(f"got: {phone} (id={act_id})")
            return PhoneNumber(number=phone, activation_id=act_id, service="5sim")
        if resp.status_code in (400, 404):
            raise NoNumberAvailable(resp.text[:100])
        raise SmsError(f"get_number failed: {resp.status_code} {resp.text[:100]}")

    def wait_for_otp(self, activation_id: str, timeout: int = POLL_TIMEOUT) -> str:
        self._log(f"polling OTP (id={activation_id})")
        deadline = time.time() + timeout
        while time.time() < deadline:
            time.sleep(POLL_INTERVAL)
            resp = self._client.get(f"{self.BASE}/user/check/{activation_id}")
            resp.raise_for_status()
            data = resp.json()
            status = data.get("status", "")
            sms_list = data.get("sms", [])
            self._log(f"  status={status} sms={len(sms_list)}")
            if sms_list:
                text = sms_list[-1].get("text", "")
                import re
                m = re.search(r"\b(\d{6})\b", text)
                if m:
                    code = m.group(1)
                    self._log(f"  OTP: {code}")
                    return code
            if status in ("CANCELED", "BANNED", "TIMEOUT"):
                raise SmsError(f"Activation {status}")
        raise SmsError(f"OTP not received within {timeout}s")

    def finish(self, activation_id: str) -> None:
        self._client.get(f"{self.BASE}/user/finish/{activation_id}")

    def cancel(self, activation_id: str) -> None:
        self._client.get(f"{self.BASE}/user/cancel/{activation_id}")

    def close(self) -> None:
        self._client.close()


# ---------------------------------------------------------------------------
# Manual (interactive) SMS
# ---------------------------------------------------------------------------

class ManualSmsClient:
    """User provides the phone number; OTP is entered interactively."""

    def __init__(self, phone: str, verbose: bool = False):
        self._phone = phone.lstrip("+").replace(" ", "").replace("-", "")
        self._verbose = verbose

    def get_number(self, **_) -> PhoneNumber:
        print(f"[manual-sms] Using phone: +{self._phone}", flush=True)
        return PhoneNumber(number=self._phone, activation_id="manual", service="manual")

    def wait_for_otp(self, activation_id: str, timeout: int = POLL_TIMEOUT) -> str:
        code = input(f"[manual-sms] Enter the 6-digit OTP sent to +{self._phone}: ").strip()
        if not code.isdigit() or len(code) != 6:
            raise SmsError(f"Invalid OTP: {code!r}")
        return code

    def finish(self, activation_id: str) -> None:
        pass

    def cancel(self, activation_id: str) -> None:
        pass

    def close(self) -> None:
        pass


# ---------------------------------------------------------------------------
# Factory
# ---------------------------------------------------------------------------

def make_client(
    service: str,
    api_key: str = "",
    phone: str = "",
    verbose: bool = False,
):
    """Create the appropriate SMS client."""
    if service == "smspool":
        return SmsPoolClient(api_key, verbose=verbose)
    elif service == "5sim":
        return FiveSimClient(api_key, verbose=verbose)
    elif service == "manual":
        return ManualSmsClient(phone, verbose=verbose)
    else:
        raise ValueError(f"Unknown SMS service: {service!r}. Choose: smspool, 5sim, manual")
