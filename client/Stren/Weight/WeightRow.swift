import SwiftUI

/// Single row for the weight list. Mirrors the web's
/// `WeightRow` template (`internal/views/weight/list.templ:206`)
/// — date + weight always, photo thumbnail and notes when
/// present. Tapping the row opens the edit sheet so the
/// user has a single affordance for both viewing and
/// editing.
struct WeightRow: View {
    let entry: WeightEntryDTO
    let weightUnit: String

    /// Tapping the row opens the edit sheet.
    let onTap: () -> Void

    /// UK-formatted date (DD/MM/YY) — matches the web's
    /// `FormattedDate` so the iOS view reads identically to
    /// the web table.
    private var formattedDate: String {
        entry.createdAt.formatted(.dateTime
            .day(.twoDigits)
            .month(.twoDigits)
            .year(.twoDigits)
        )
    }

    var body: some View {
        HStack(spacing: DSSpacing.sm) {
            VStack(alignment: .leading, spacing: 2) {
                Text(formattedDate)
                    .font(.subheadline.weight(.medium))
                    .foregroundStyle(DSColors.text)
                Text(entry.formattedWeight(in: weightUnit))
                    .font(.body)
                    .foregroundStyle(DSColors.text)
                if !entry.notes.isEmpty {
                    Text(entry.notes)
                        .font(.caption)
                        .foregroundStyle(DSColors.textSecondary)
                        .lineLimit(2)
                }
            }

            Spacer()

            if entry.hasPhoto, let url = URL(string: entry.photoURL) {
                AsyncImage(url: url) { phase in
                    switch phase {
                    case .success(let image):
                        image
                            .resizable()
                            .scaledToFill()
                    case .failure, .empty:
                        photoPlaceholder
                    @unknown default:
                        photoPlaceholder
                    }
                }
                .frame(width: 40, height: 40)
                .clipShape(RoundedRectangle(cornerRadius: DSSpacing.cornerRadiusSmall, style: .continuous))
            }
        }
        .padding(.vertical, DSSpacing.xs)
        .contentShape(Rectangle())
        .onTapGesture {
            onTap()
        }
    }

    private var photoPlaceholder: some View {
        RoundedRectangle(cornerRadius: DSSpacing.cornerRadiusSmall, style: .continuous)
            .fill(DSColors.surfaceElevated)
            .overlay(
                Image(systemName: "photo")
                    .foregroundStyle(DSColors.textSecondary)
            )
    }
}
