from typing import List, Optional

import cv2
import numpy as np


def compute_ssim_pair(frames_a: List[np.ndarray], frames_b: List[np.ndarray]) -> Optional[float]:
    """
    Compare two lists of BGR frames and return average SSIM as a 0–100 score.
    Frame counts don't need to match; we compare min(len_a, len_b) pairs.
    Returns None if either list is empty.
    """
    if not frames_a or not frames_b:
        return None

    n = min(len(frames_a), len(frames_b))
    scores: List[float] = []

    for i in range(n):
        gray_a = cv2.cvtColor(frames_a[i], cv2.COLOR_BGR2GRAY).astype(np.float32)
        gray_b = cv2.cvtColor(frames_b[i], cv2.COLOR_BGR2GRAY).astype(np.float32)
        s = _ssim(gray_a, gray_b)
        scores.append(s)

    if not scores:
        return None

    avg = sum(scores) / len(scores)
    return round(float(np.clip(avg * 100, 0.0, 100.0)), 1)


def _ssim(img1: np.ndarray, img2: np.ndarray) -> float:
    """Compute SSIM between two single-channel float32 images (values 0–255)."""
    C1 = (0.01 * 255) ** 2
    C2 = (0.03 * 255) ** 2
    k = (11, 11)
    sigma = 1.5

    mu1 = cv2.GaussianBlur(img1, k, sigma)
    mu2 = cv2.GaussianBlur(img2, k, sigma)
    mu1_sq = mu1 * mu1
    mu2_sq = mu2 * mu2
    mu1_mu2 = mu1 * mu2

    sigma1_sq = cv2.GaussianBlur(img1 * img1, k, sigma) - mu1_sq
    sigma2_sq = cv2.GaussianBlur(img2 * img2, k, sigma) - mu2_sq
    sigma12 = cv2.GaussianBlur(img1 * img2, k, sigma) - mu1_mu2

    numerator = (2 * mu1_mu2 + C1) * (2 * sigma12 + C2)
    denominator = (mu1_sq + mu2_sq + C1) * (sigma1_sq + sigma2_sq + C2)
    ssim_map = numerator / (denominator + 1e-10)
    return float(ssim_map.mean())
