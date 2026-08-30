cbuffer GlobalConstants : register(b0)
{
    float2 u_viewport_size;
    float2 u_pad;
};

struct VSInput
{
    uint vertex_id : SV_VertexID;
    uint instance_id : SV_InstanceID;
    // Instance data
    float4 bounds : INST_BOUNDS;           // min_x, min_y, max_x, max_y
    float4 content_mask : INST_MASK;       // min_x, min_y, max_x, max_y
    float4 bg_color : INST_BG_COLOR;       // premultiplied rgba
    float4 border_color : INST_BORDER_COL; // premultiplied rgba
    float4 corner_radii : INST_RADII;      // top_left, top_right, bottom_right, bottom_left
    float4 border_widths : INST_BORDER_W;  // top, right, bottom, left
    uint border_style : INST_BORDER_STYLE; // 0 = solid, 1 = dashed
    uint pad0 : INST_PAD0;
    uint pad1 : INST_PAD1;
    uint pad2 : INST_PAD2;
};

struct PSInput
{
    float4 position : SV_POSITION;
    float2 screen_pos : SCREEN_POS;
    float4 bounds : BOUNDS;
    float4 content_mask : CONTENT_MASK;
    float4 bg_color : BG_COLOR;
    float4 border_color : BORDER_COLOR;
    float4 corner_radii : CORNER_RADII;
    float4 border_widths : BORDER_WIDTHS;
    uint border_style : BORDER_STYLE;
};

static const float2 QUAD_CORNERS[6] = {
    float2(0.0, 0.0),
    float2(1.0, 0.0),
    float2(0.0, 1.0),
    float2(0.0, 1.0),
    float2(1.0, 0.0),
    float2(1.0, 1.0)
};

PSInput vs_main(VSInput input)
{
    PSInput output;
    float2 corner = QUAD_CORNERS[input.vertex_id % 6];
    float2 pos = float2(
        lerp(input.bounds.x, input.bounds.z, corner.x),
        lerp(input.bounds.y, input.bounds.w, corner.y)
    );

    float2 ndc = float2(
        (pos.x / u_viewport_size.x) * 2.0 - 1.0,
        1.0 - (pos.y / u_viewport_size.y) * 2.0
    );

    output.position = float4(ndc, 0.0, 1.0);
    output.screen_pos = pos;
    output.bounds = input.bounds;
    output.content_mask = input.content_mask;
    output.bg_color = input.bg_color;
    output.border_color = input.border_color;
    output.corner_radii = input.corner_radii;
    output.border_widths = input.border_widths;
    output.border_style = input.border_style;
    return output;
}

float rounded_box_sdf(float2 p, float2 half_size, float4 radii)
{
    float r = (p.x < 0.0) ? ((p.y < 0.0) ? radii.x : radii.w) : ((p.y < 0.0) ? radii.y : radii.z);
    r = min(r, min(half_size.x, half_size.y));
    float2 q = abs(p) - half_size + float2(r, r);
    return length(max(q, 0.0)) + min(max(q.x, q.y), 0.0) - r;
}

float4 ps_main(PSInput input) : SV_TARGET
{
    // Content mask clip
    if (input.content_mask.z > input.content_mask.x && input.content_mask.w > input.content_mask.y)
    {
        if (input.screen_pos.x < input.content_mask.x || input.screen_pos.x > input.content_mask.z ||
            input.screen_pos.y < input.content_mask.y || input.screen_pos.y > input.content_mask.w)
        {
            discard;
        }
    }

    float2 size = float2(input.bounds.z - input.bounds.x, input.bounds.w - input.bounds.y);
    float2 half_size = size * 0.5;
    float2 centre = float2(input.bounds.x + half_size.x, input.bounds.y + half_size.y);
    float2 p = input.screen_pos - centre;

    float dist_outer = rounded_box_sdf(p, half_size, input.corner_radii);
    float alpha_outer = saturate(0.5 - dist_outer);

    // If no border, render background only
    float max_border = max(max(input.border_widths.x, input.border_widths.y),
                           max(input.border_widths.z, input.border_widths.w));
    if (max_border <= 0.0)
    {
        return input.bg_color * alpha_outer;
    }

    float2 inner_half_size = max(float2(0.0, 0.0), half_size - float2(
        (input.border_widths.w + input.border_widths.y) * 0.5,
        (input.border_widths.x + input.border_widths.z) * 0.5
    ));
    float2 inner_offset = float2(
        (input.border_widths.w - input.border_widths.y) * 0.5,
        (input.border_widths.x - input.border_widths.z) * 0.5
    );
    float4 inner_radii = max(float4(0.0, 0.0, 0.0, 0.0), input.corner_radii - float4(
        max(input.border_widths.x, input.border_widths.w),
        max(input.border_widths.x, input.border_widths.y),
        max(input.border_widths.z, input.border_widths.y),
        max(input.border_widths.z, input.border_widths.w)
    ));

    float dist_inner = rounded_box_sdf(p - inner_offset, inner_half_size, inner_radii);
    float alpha_inner = saturate(0.5 - dist_inner);
    float alpha_border = saturate(alpha_outer - alpha_inner);

    if (input.border_style == 1) // Dashed
    {
        float perimeter_dist = input.screen_pos.x + input.screen_pos.y;
        float dash = fmod(perimeter_dist, 16.0);
        if (dash > 8.0)
        {
            alpha_border = 0.0;
        }
    }

    return input.bg_color * alpha_inner + input.border_color * alpha_border;
}
